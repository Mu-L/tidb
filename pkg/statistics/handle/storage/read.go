// Copyright 2023 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/failpoint"
	"github.com/pingcap/tidb/pkg/config"
	"github.com/pingcap/tidb/pkg/infoschema"
	"github.com/pingcap/tidb/pkg/kv"
	"github.com/pingcap/tidb/pkg/meta/model"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pingcap/tidb/pkg/sessionctx"
	"github.com/pingcap/tidb/pkg/statistics"
	"github.com/pingcap/tidb/pkg/statistics/asyncload"
	statslogutil "github.com/pingcap/tidb/pkg/statistics/handle/logutil"
	statstypes "github.com/pingcap/tidb/pkg/statistics/handle/types"
	"github.com/pingcap/tidb/pkg/statistics/handle/util"
	"github.com/pingcap/tidb/pkg/types"
	"github.com/pingcap/tidb/pkg/util/chunk"
	"github.com/pingcap/tidb/pkg/util/intest"
	"github.com/pingcap/tidb/pkg/util/logutil"
	"github.com/pingcap/tidb/pkg/util/memory"
	"github.com/pingcap/tidb/pkg/util/sqlexec"
	"go.uber.org/zap"
)

// StatsMetaCountAndModifyCount reads count and modify_count for the given table from mysql.stats_meta.
func StatsMetaCountAndModifyCount(
	ctx context.Context,
	sctx sessionctx.Context,
	tableID int64,
) (count, modifyCount int64, isNull bool, err error) {
	return statsMetaCountAndModifyCount(ctx, sctx, tableID, false)
}

// StatsMetaCountAndModifyCountForUpdate reads count and modify_count for the given table from mysql.stats_meta with lock.
func StatsMetaCountAndModifyCountForUpdate(
	ctx context.Context,
	sctx sessionctx.Context,
	tableID int64,
) (count, modifyCount int64, isNull bool, err error) {
	return statsMetaCountAndModifyCount(ctx, sctx, tableID, true)
}

func statsMetaCountAndModifyCount(
	ctx context.Context,
	sctx sessionctx.Context,
	tableID int64,
	forUpdate bool,
) (count, modifyCount int64, isNull bool, err error) {
	sql := "select count, modify_count from mysql.stats_meta where table_id = %?"
	if forUpdate {
		sql += " for update"
	}
	rows, _, err := util.ExecRowsWithCtx(ctx, sctx, sql, tableID)
	if err != nil {
		return 0, 0, false, err
	}
	if len(rows) == 0 {
		return 0, 0, true, nil
	}
	count = int64(rows[0].GetUint64(0))
	modifyCount = rows[0].GetInt64(1)
	return count, modifyCount, false, nil
}

// HistMetaFromStorageWithHighPriority reads the meta info of the histogram from the storage.
func HistMetaFromStorageWithHighPriority(sctx sessionctx.Context, item *model.TableItemID, possibleColInfo *model.ColumnInfo) (*statistics.Histogram, int64, error) {
	isIndex := 0
	var tp *types.FieldType
	if item.IsIndex {
		isIndex = 1
		tp = types.NewFieldType(mysql.TypeBlob)
	} else {
		tp = &possibleColInfo.FieldType
	}
	rows, _, err := util.ExecRows(sctx,
		"select high_priority distinct_count, version, null_count, tot_col_size, stats_ver, correlation from mysql.stats_histograms where table_id = %? and hist_id = %? and is_index = %?",
		item.TableID,
		item.ID,
		isIndex,
	)
	if err != nil {
		return nil, 0, err
	}
	if len(rows) == 0 {
		return nil, 0, nil
	}
	hist := statistics.NewHistogram(item.ID, rows[0].GetInt64(0), rows[0].GetInt64(2), rows[0].GetUint64(1), tp, chunk.InitialCapacity, rows[0].GetInt64(3))
	hist.Correlation = rows[0].GetFloat64(5)
	return hist, rows[0].GetInt64(4), nil
}

// HistogramFromStorageWithPriority wraps the HistogramFromStorage with the given kv.Priority.
// Sync load and async load will use high priority to get data.
func HistogramFromStorageWithPriority(
	sctx sessionctx.Context,
	tableID int64,
	colID int64,
	tp *types.FieldType,
	distinct int64,
	isIndex int,
	ver uint64,
	nullCount int64,
	totColSize int64,
	corr float64,
	priority int,
) (*statistics.Histogram, error) {
	selectPrefix := "select "
	switch priority {
	case kv.PriorityHigh:
		selectPrefix += "high_priority "
	case kv.PriorityLow:
		selectPrefix += "low_priority "
	}
	rows, fields, err := util.ExecRows(sctx, selectPrefix+"count, repeats, lower_bound, upper_bound, ndv from mysql.stats_buckets where table_id = %? and is_index = %? and hist_id = %? order by bucket_id", tableID, isIndex, colID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	bucketSize := len(rows)
	hg := statistics.NewHistogram(colID, distinct, nullCount, ver, tp, bucketSize, totColSize)
	hg.Correlation = corr
	totalCount := int64(0)
	for i := range bucketSize {
		count := rows[i].GetInt64(0)
		repeats := rows[i].GetInt64(1)
		var upperBound, lowerBound types.Datum
		if isIndex == 1 {
			lowerBound = rows[i].GetDatum(2, &fields[2].Column.FieldType)
			upperBound = rows[i].GetDatum(3, &fields[3].Column.FieldType)
		} else {
			d := rows[i].GetDatum(2, &fields[2].Column.FieldType)
			// For new collation data, when storing the bounds of the histogram, we store the collate key instead of the
			// original value.
			// But there's additional conversion logic for new collation data, and the collate key might be longer than
			// the FieldType.flen.
			// If we use the original FieldType here, there might be errors like "Invalid utf8mb4 character string"
			// or "Data too long".
			// So we change it to TypeBlob to bypass those logics here.
			if tp.EvalType() == types.ETString && tp.GetType() != mysql.TypeEnum && tp.GetType() != mysql.TypeSet {
				tp = types.NewFieldType(mysql.TypeBlob)
			}
			lowerBound, err = convertBoundFromBlob(statistics.UTCWithAllowInvalidDateCtx, d, tp)
			if err != nil {
				return nil, errors.Trace(err)
			}
			d = rows[i].GetDatum(3, &fields[3].Column.FieldType)
			upperBound, err = convertBoundFromBlob(statistics.UTCWithAllowInvalidDateCtx, d, tp)
			if err != nil {
				return nil, errors.Trace(err)
			}
		}
		totalCount += count
		hg.AppendBucketWithNDV(&lowerBound, &upperBound, totalCount, repeats, rows[i].GetInt64(4))
	}
	hg.PreCalculateScalar()
	return hg, nil
}

// CMSketchAndTopNFromStorageWithHighPriority reads CMSketch and TopN from storage.
func CMSketchAndTopNFromStorageWithHighPriority(sctx sessionctx.Context, tblID int64, isIndex, histID, statsVer int64) (_ *statistics.CMSketch, _ *statistics.TopN, err error) {
	topNRows, _, err := util.ExecRows(sctx, "select HIGH_PRIORITY value, count from mysql.stats_top_n where table_id = %? and is_index = %? and hist_id = %?", tblID, isIndex, histID)
	if err != nil {
		return nil, nil, err
	}
	// If we are on version higher than 1. Don't read Count-Min Sketch.
	if statsVer > statistics.Version1 {
		return statistics.DecodeCMSketchAndTopN(nil, topNRows)
	}
	rows, _, err := util.ExecRows(sctx, "select cm_sketch from mysql.stats_histograms where table_id = %? and is_index = %? and hist_id = %?", tblID, isIndex, histID)
	if err != nil {
		return nil, nil, err
	}
	if len(rows) == 0 {
		return statistics.DecodeCMSketchAndTopN(nil, topNRows)
	}
	return statistics.DecodeCMSketchAndTopN(rows[0].GetBytes(0), topNRows)
}

// CMSketchFromStorage reads CMSketch from storage
func CMSketchFromStorage(sctx sessionctx.Context, tblID int64, isIndex int, histID int64) (_ *statistics.CMSketch, err error) {
	rows, _, err := util.ExecRows(sctx, "select cm_sketch from mysql.stats_histograms where table_id = %? and is_index = %? and hist_id = %?", tblID, isIndex, histID)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return statistics.DecodeCMSketch(rows[0].GetBytes(0))
}

// TopNFromStorage reads TopN from storage
func TopNFromStorage(sctx sessionctx.Context, tblID int64, isIndex int, histID int64) (_ *statistics.TopN, err error) {
	rows, _, err := util.ExecRows(sctx, "select HIGH_PRIORITY value, count from mysql.stats_top_n where table_id = %? and is_index = %? and hist_id = %?", tblID, isIndex, histID)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return statistics.DecodeTopN(rows), nil
}

// FMSketchFromStorage reads FMSketch from storage
func FMSketchFromStorage(sctx sessionctx.Context, tblID int64, isIndex, histID int64) (_ *statistics.FMSketch, err error) {
	rows, _, err := util.ExecRows(sctx, "select value from mysql.stats_fm_sketch where table_id = %? and is_index = %? and hist_id = %?", tblID, isIndex, histID)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return statistics.DecodeFMSketch(rows[0].GetBytes(0))
}

// CheckSkipPartition checks if we can skip loading the partition.
func CheckSkipPartition(sctx sessionctx.Context, tblID int64, isIndex int) error {
	rows, _, err := util.ExecRows(sctx, "select distinct_count from mysql.stats_histograms where table_id =%? and is_index = %?", tblID, isIndex)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return types.ErrPartitionStatsMissing
	}
	return nil
}

// CheckSkipColumnPartiion checks if we can skip loading the partition.
func CheckSkipColumnPartiion(sctx sessionctx.Context, tblID int64, isIndex int, histsID int64) error {
	rows, _, err := util.ExecRows(sctx, "select distinct_count from mysql.stats_histograms where table_id = %? and is_index = %? and hist_id = %?", tblID, isIndex, histsID)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return types.ErrPartitionColumnStatsMissing
	}
	return nil
}

// ExtendedStatsFromStorage reads extended stats from storage.
func ExtendedStatsFromStorage(sctx sessionctx.Context, table *statistics.Table, tableID int64, loadAll bool) (*statistics.Table, error) {
	failpoint.Inject("injectExtStatsLoadErr", func() {
		failpoint.Return(nil, errors.New("gofail extendedStatsFromStorage error"))
	})
	lastVersion := uint64(0)
	if table.ExtendedStats != nil && !loadAll {
		lastVersion = table.ExtendedStats.LastUpdateVersion
	} else {
		table.ExtendedStats = statistics.NewExtendedStatsColl()
	}
	rows, _, err := util.ExecRows(sctx, "select name, status, type, column_ids, stats, version from mysql.stats_extended where table_id = %? and status in (%?, %?, %?) and version > %?",
		tableID, statistics.ExtendedStatsInited, statistics.ExtendedStatsAnalyzed, statistics.ExtendedStatsDeleted, lastVersion)
	if err != nil || len(rows) == 0 {
		return table, nil
	}
	for _, row := range rows {
		lastVersion = max(lastVersion, row.GetUint64(5))
		name := row.GetString(0)
		status := uint8(row.GetInt64(1))
		if status == statistics.ExtendedStatsDeleted || status == statistics.ExtendedStatsInited {
			delete(table.ExtendedStats.Stats, name)
		} else {
			item := &statistics.ExtendedStatsItem{
				Tp: uint8(row.GetInt64(2)),
			}
			colIDs := row.GetString(3)
			err := json.Unmarshal([]byte(colIDs), &item.ColIDs)
			if err != nil {
				statslogutil.StatsLogger().Error("decode column IDs failed", zap.String("column_ids", colIDs), zap.Error(err))
				return nil, err
			}
			statsStr := row.GetString(4)
			if item.Tp == ast.StatsTypeCardinality || item.Tp == ast.StatsTypeCorrelation {
				if statsStr != "" {
					item.ScalarVals, err = strconv.ParseFloat(statsStr, 64)
					if err != nil {
						statslogutil.StatsLogger().Error("parse scalar stats failed", zap.String("stats", statsStr), zap.Error(err))
						return nil, err
					}
				}
			} else {
				item.StringVals = statsStr
			}
			table.ExtendedStats.Stats[name] = item
		}
	}
	table.ExtendedStats.LastUpdateVersion = lastVersion
	return table, nil
}

func indexStatsFromStorage(sctx sessionctx.Context, row chunk.Row, table *statistics.Table, tableInfo *model.TableInfo, loadAll bool, lease time.Duration, tracker *memory.Tracker) error {
	histID := row.GetInt64(2)
	distinct := row.GetInt64(3)
	histVer := row.GetUint64(4)
	nullCount := row.GetInt64(5)
	statsVer := row.GetInt64(7)
	idx := table.GetIdx(histID)

	for _, idxInfo := range tableInfo.Indices {
		if histID != idxInfo.ID {
			continue
		}
		table.ColAndIdxExistenceMap.InsertIndex(idxInfo.ID, statsVer != statistics.Version0)
		// All the objects in the table shares the same stats version.
		// Update here.
		if statsVer != statistics.Version0 {
			table.StatsVer = int(statsVer)
			table.LastAnalyzeVersion = max(table.LastAnalyzeVersion, histVer)
		}
		// We will not load buckets, topn and cmsketch if:
		// 1. lease > 0, and:
		// 2. the index doesn't have any of buckets, topn, cmsketch in memory before, and:
		// 3. loadAll is false.
		// 4. lite-init-stats is true(remove the condition when lite init stats is GA).
		notNeedLoad := lease > 0 &&
			(idx == nil || ((!idx.IsStatsInitialized() || idx.IsAllEvicted()) && idx.LastUpdateVersion < histVer)) &&
			!loadAll &&
			config.GetGlobalConfig().Performance.LiteInitStats
		if notNeedLoad {
			// If we don't have this index in memory, skip it.
			if idx == nil {
				return nil
			}
			idx = &statistics.Index{
				Histogram:  *statistics.NewHistogram(histID, distinct, nullCount, histVer, types.NewFieldType(mysql.TypeBlob), 0, 0),
				StatsVer:   statsVer,
				Info:       idxInfo,
				PhysicalID: table.PhysicalID,
			}
			if idx.IsAnalyzed() {
				idx.StatsLoadedStatus = statistics.NewStatsAllEvictedStatus()
			}
			break
		}
		if idx == nil || idx.LastUpdateVersion < histVer || loadAll {
			hg, err := HistogramFromStorageWithPriority(sctx, table.PhysicalID, histID, types.NewFieldType(mysql.TypeBlob), distinct, 1, histVer, nullCount, 0, 0, kv.PriorityNormal)
			if err != nil {
				return errors.Trace(err)
			}
			cms, topN, err := CMSketchAndTopNFromStorageWithHighPriority(sctx, table.PhysicalID, 1, idxInfo.ID, statsVer)
			if err != nil {
				return errors.Trace(err)
			}
			var fmSketch *statistics.FMSketch
			if loadAll {
				// FMSketch is only used when merging partition stats into global stats. When merging partition stats into global stats,
				// we load all the statistics, i.e., loadAll is true.
				fmSketch, err = FMSketchFromStorage(sctx, table.PhysicalID, 1, histID)
				if err != nil {
					return errors.Trace(err)
				}
			}
			idx = &statistics.Index{
				Histogram:  *hg,
				CMSketch:   cms,
				TopN:       topN,
				FMSketch:   fmSketch,
				Info:       idxInfo,
				StatsVer:   statsVer,
				PhysicalID: table.PhysicalID,
			}
			if statsVer != statistics.Version0 {
				idx.StatsLoadedStatus = statistics.NewStatsFullLoadStatus()
			}
		}
		break
	}
	if idx != nil {
		if tracker != nil {
			tracker.Consume(idx.MemoryUsage().TotalMemoryUsage())
		}
		table.SetIdx(histID, idx)
	} else {
		logutil.BgLogger().Debug("we cannot find index id in table info. It may be deleted.", zap.Int64("indexID", histID), zap.String("table", tableInfo.Name.O))
	}
	return nil
}

func columnStatsFromStorage(sctx sessionctx.Context, row chunk.Row, table *statistics.Table, tableInfo *model.TableInfo, loadAll bool, lease time.Duration, tracker *memory.Tracker) error {
	histID := row.GetInt64(2)
	distinct := row.GetInt64(3)
	histVer := row.GetUint64(4)
	nullCount := row.GetInt64(5)
	totColSize := row.GetInt64(6)
	statsVer := row.GetInt64(7)
	correlation := row.GetFloat64(8)
	col := table.GetCol(histID)

	for _, colInfo := range tableInfo.Columns {
		if histID != colInfo.ID {
			continue
		}
		table.ColAndIdxExistenceMap.InsertCol(histID, statsVer != statistics.Version0 || distinct > 0 || nullCount > 0)
		// All the objects in the table shares the same stats version.
		// Update here.
		if statsVer != statistics.Version0 {
			table.StatsVer = int(statsVer)
			table.LastAnalyzeVersion = max(table.LastAnalyzeVersion, histVer)
		}
		isHandle := tableInfo.PKIsHandle && mysql.HasPriKeyFlag(colInfo.GetFlag())
		// We will not load buckets, topn and cmsketch if:
		// 1. lease > 0, and:
		// 2. this column is not handle or lite-init-stats is true(remove the condition when lite init stats is GA), and:
		// 3. the column doesn't have any of buckets, topn, cmsketch in memory before, and:
		// 4. loadAll is false.
		//
		// Here is the explanation of the condition `!col.IsStatsInitialized() || col.IsAllEvicted()`.
		// For one column:
		// 1. If there is no stats for it in the storage(i.e., analyze has never been executed before), then its stats status
		//    would be `!col.IsStatsInitialized()`. In this case we should go the `notNeedLoad` path.
		// 2. If there exists stats for it in the storage but its stats status is `col.IsAllEvicted()`, there are two
		//    sub cases for this case. One is that the column stats have never been used/needed by the optimizer so they have
		//    never been loaded. The other is that the column stats were loaded and then evicted. For the both sub cases,
		//    we should go the `notNeedLoad` path.
		// 3. If some parts(Histogram/TopN/CMSketch) of stats for it exist in TiDB memory currently, we choose to load all of
		//    its new stats once we find stats version is updated.
		notNeedLoad := lease > 0 &&
			(!isHandle || config.GetGlobalConfig().Performance.LiteInitStats) &&
			(col == nil || ((!col.IsStatsInitialized() || col.IsAllEvicted()) && col.LastUpdateVersion < histVer)) &&
			!loadAll
		if notNeedLoad {
			// If we don't have the column in memory currently, just skip it.
			if col == nil {
				return nil
			}
			col = &statistics.Column{
				PhysicalID: table.PhysicalID,
				Histogram:  *statistics.NewHistogram(histID, distinct, nullCount, histVer, &colInfo.FieldType, 0, totColSize),
				Info:       colInfo,
				IsHandle:   tableInfo.PKIsHandle && mysql.HasPriKeyFlag(colInfo.GetFlag()),
				StatsVer:   statsVer,
			}
			if col.StatsAvailable() {
				col.StatsLoadedStatus = statistics.NewStatsAllEvictedStatus()
			}
			col.Histogram.Correlation = correlation
			break
		}
		if col == nil || col.LastUpdateVersion < histVer || loadAll {
			hg, err := HistogramFromStorageWithPriority(sctx, table.PhysicalID, histID, &colInfo.FieldType, distinct, 0, histVer, nullCount, totColSize, correlation, kv.PriorityNormal)
			if err != nil {
				return errors.Trace(err)
			}
			cms, topN, err := CMSketchAndTopNFromStorageWithHighPriority(sctx, table.PhysicalID, 0, colInfo.ID, statsVer)
			if err != nil {
				return errors.Trace(err)
			}
			var fmSketch *statistics.FMSketch
			if loadAll {
				// FMSketch is only used when merging partition stats into global stats. When merging partition stats into global stats,
				// we load all the statistics, i.e., loadAll is true.
				fmSketch, err = FMSketchFromStorage(sctx, table.PhysicalID, 0, histID)
				if err != nil {
					return errors.Trace(err)
				}
			}
			col = &statistics.Column{
				PhysicalID: table.PhysicalID,
				Histogram:  *hg,
				Info:       colInfo,
				CMSketch:   cms,
				TopN:       topN,
				FMSketch:   fmSketch,
				IsHandle:   tableInfo.PKIsHandle && mysql.HasPriKeyFlag(colInfo.GetFlag()),
				StatsVer:   statsVer,
			}
			if col.StatsAvailable() {
				col.StatsLoadedStatus = statistics.NewStatsFullLoadStatus()
			}
			break
		}
		if col.TotColSize != totColSize {
			newCol := *col
			newCol.TotColSize = totColSize
			col = &newCol
		}
		break
	}
	if col != nil {
		if tracker != nil {
			tracker.Consume(col.MemoryUsage().TotalMemoryUsage())
		}
		table.SetCol(col.ID, col)
	} else {
		// If we didn't find a Column or Index in tableInfo, we won't load the histogram for it.
		// But don't worry, next lease the ddl will be updated, and we will load a same table for two times to
		// avoid error.
		logutil.BgLogger().Debug("we cannot find column in table info now. It may be deleted", zap.Int64("colID", histID), zap.String("table", tableInfo.Name.O))
	}
	return nil
}

// TableStatsFromStorage loads table stats info from storage.
func TableStatsFromStorage(sctx sessionctx.Context, snapshot uint64, tableInfo *model.TableInfo, tableID int64, loadAll bool, lease time.Duration, table *statistics.Table) (_ *statistics.Table, err error) {
	tracker := memory.NewTracker(memory.LabelForAnalyzeMemory, -1)
	tracker.AttachTo(sctx.GetSessionVars().MemTracker)
	defer tracker.Detach()
	// If table stats is pseudo, we also need to copy it, since we will use the column stats when
	// the average error rate of it is small.
	if table == nil || snapshot > 0 {
		histColl := *statistics.NewHistColl(tableID, 0, 0, 4, 4)
		table = &statistics.Table{
			HistColl:              histColl,
			ColAndIdxExistenceMap: statistics.NewColAndIndexExistenceMap(len(tableInfo.Columns), len(tableInfo.Indices)),
		}
	} else {
		// We copy it before writing to avoid race.
		table = table.Copy()
	}
	table.Pseudo = false

	realtimeCount, modidyCount, isNull, err := StatsMetaCountAndModifyCount(util.StatsCtx, sctx, tableID)
	if err != nil || isNull {
		return nil, err
	}
	table.ModifyCount = modidyCount
	table.RealtimeCount = realtimeCount

	rows, _, err := util.ExecRows(sctx, "select table_id, is_index, hist_id, distinct_count, version, null_count, tot_col_size, stats_ver, correlation from mysql.stats_histograms where table_id = %?", tableID)
	if err != nil {
		return nil, err
	}
	// Check table has no index/column stats.
	if len(rows) == 0 {
		return table, nil
	}
	for _, row := range rows {
		if err := sctx.GetSessionVars().SQLKiller.HandleSignal(); err != nil {
			return nil, err
		}
		if row.GetInt64(1) > 0 {
			err = indexStatsFromStorage(sctx, row, table, tableInfo, loadAll, lease, tracker)
		} else {
			err = columnStatsFromStorage(sctx, row, table, tableInfo, loadAll, lease, tracker)
		}
		if err != nil {
			return nil, err
		}
	}
	// If DROP STATS executes, we need to reset the stats version to 0.
	if table.StatsVer != statistics.Version0 {
		allZero := true
		table.ForEachColumnImmutable(func(_ int64, c *statistics.Column) bool {
			if c.StatsVer != statistics.Version0 {
				allZero = false
				return true
			}
			return false
		})
		table.ForEachIndexImmutable(func(_ int64, idx *statistics.Index) bool {
			if idx.StatsVer != statistics.Version0 {
				allZero = false
				return true
			}
			return false
		})
		if allZero {
			table.StatsVer = statistics.Version0
		}
	}
	table.ColAndIdxExistenceMap.SetChecked()
	return ExtendedStatsFromStorage(sctx, table, tableID, loadAll)
}

// LoadHistogram will load histogram from storage.
func LoadHistogram(sctx sessionctx.Context, tableID int64, isIndex int, histID int64, tableInfo *model.TableInfo) (*statistics.Histogram, error) {
	row, _, err := util.ExecRows(sctx, "select distinct_count, version, null_count, tot_col_size, stats_ver, flag, correlation from mysql.stats_histograms where table_id = %? and is_index = %? and hist_id = %?", tableID, isIndex, histID)
	if err != nil || len(row) == 0 {
		return nil, err
	}
	distinct := row[0].GetInt64(0)
	histVer := row[0].GetUint64(1)
	nullCount := row[0].GetInt64(2)
	var totColSize int64
	var corr float64
	var tp types.FieldType
	if isIndex == 0 {
		totColSize = row[0].GetInt64(3)
		corr = row[0].GetFloat64(6)
		for _, colInfo := range tableInfo.Columns {
			if histID != colInfo.ID {
				continue
			}
			tp = colInfo.FieldType
			break
		}
		return HistogramFromStorageWithPriority(sctx, tableID, histID, &tp, distinct, isIndex, histVer, nullCount, totColSize, corr, kv.PriorityNormal)
	}
	return HistogramFromStorageWithPriority(sctx, tableID, histID, types.NewFieldType(mysql.TypeBlob), distinct, isIndex, histVer, nullCount, 0, 0, kv.PriorityNormal)
}

// LoadNeededHistograms will load histograms for those needed columns/indices.
func LoadNeededHistograms(sctx sessionctx.Context, is infoschema.InfoSchema, statsHandle statstypes.StatsHandle, loadFMSketch bool) (err error) {
	items := asyncload.AsyncLoadHistogramNeededItems.AllItems()
	for _, item := range items {
		if !item.IsIndex {
			err = loadNeededColumnHistograms(sctx, statsHandle, item.TableItemID, loadFMSketch, item.FullLoad)
		} else {
			// Index is always full load.
			err = loadNeededIndexHistograms(sctx, is, statsHandle, item.TableItemID, loadFMSketch)
		}
		if err != nil {
			statslogutil.StatsSampleLogger().Error("load needed histogram failed",
				zap.Error(err),
				zap.Int64("tableID", item.TableID),
				zap.Int64("histID", item.ID),
				zap.Bool("isIndex", item.IsIndex),
				zap.Bool("IsSyncLoadFailed", item.IsSyncLoadFailed),
				zap.Bool("fullLoad", item.FullLoad),
			)
			intest.Assert(false, "load needed histogram failed")
			return errors.Trace(err)
		}
	}
	return nil
}

// CleanFakeItemsForShowHistInFlights cleans the invalid inserted items.
func CleanFakeItemsForShowHistInFlights(statsCache statstypes.StatsCache) int {
	items := asyncload.AsyncLoadHistogramNeededItems.AllItems()
	reallyNeeded := 0
	for _, item := range items {
		tbl, ok := statsCache.Get(item.TableID)
		if !ok {
			asyncload.AsyncLoadHistogramNeededItems.Delete(item.TableItemID)
			continue
		}
		loadNeeded := false
		if item.IsIndex {
			_, loadNeeded = tbl.IndexIsLoadNeeded(item.ID)
		} else {
			var analyzed bool
			_, loadNeeded, analyzed = tbl.ColumnIsLoadNeeded(item.ID, item.FullLoad)
			loadNeeded = loadNeeded && analyzed
		}
		if !loadNeeded {
			asyncload.AsyncLoadHistogramNeededItems.Delete(item.TableItemID)
			continue
		}
		reallyNeeded++
	}
	return reallyNeeded
}

func loadNeededColumnHistograms(sctx sessionctx.Context, statsHandle statstypes.StatsHandle, col model.TableItemID, loadFMSketch bool, fullLoad bool) (err error) {
	// Regardless of whether the load is successful or not, we must remove the item from the async load list.
	// The principle is to load the histogram for each column at most once in async load, as we already have a retry mechanism in the sync load.
	defer asyncload.AsyncLoadHistogramNeededItems.Delete(col)
	statsTbl, ok := statsHandle.Get(col.TableID)
	if !ok {
		// This could happen when the table is dropped after the async load is triggered.
		statslogutil.StatsSampleLogger().Info(
			"Table statistics item not found, possibly due to table being dropped",
			zap.Int64("tableID", col.TableID),
			zap.Int64("columnID", col.ID),
		)
		return nil
	}
	// When lite-init-stats is disabled, we cannot store the column info in the ColAndIdxExistenceMap.
	// Because we don't want to access all table information when init stats.
	// Therefore, we need to get the column info from the domain on demand.
	is := sctx.GetLatestInfoSchema().(infoschema.InfoSchema)
	tbl, ok := statsHandle.TableInfoByID(is, col.TableID)
	if !ok {
		// This could happen when the table is dropped after the async load is triggered.
		statslogutil.StatsSampleLogger().Info(
			"Table information not found, possibly due to table being dropped",
			zap.Int64("tableID", col.TableID),
			zap.Int64("columnID", col.ID),
		)
		return nil
	}
	tblInfo := tbl.Meta()
	colInfo := tblInfo.GetColumnByID(col.ID)
	if colInfo == nil {
		// This could happen when the column is dropped after the async load is triggered.
		statslogutil.StatsSampleLogger().Info(
			"Column information not found, possibly due to column being dropped",
			zap.Int64("tableID", col.TableID),
			zap.Int64("columnID", col.ID),
		)
		return nil
	}

	_, loadNeeded, analyzed := statsTbl.ColumnIsLoadNeeded(col.ID, true)
	if !loadNeeded || !analyzed {
		// If this column is not analyzed yet and we don't have it in memory.
		// We create a fake one for the pseudo estimation.
		// Otherwise, it will trigger the sync/async load again, even if the column has not been analyzed.
		if loadNeeded && !analyzed {
			fakeCol := statistics.EmptyColumn(tblInfo.ID, tblInfo.PKIsHandle, colInfo)
			statsTbl = statsTbl.Copy()
			statsTbl.SetCol(col.ID, fakeCol)
			statsHandle.UpdateStatsCache(statstypes.CacheUpdate{
				Updated: []*statistics.Table{statsTbl},
			})
		}
		return nil
	}

	hg, statsVer, err := HistMetaFromStorageWithHighPriority(sctx, &col, colInfo)
	if hg == nil || err != nil {
		if hg == nil {
			statslogutil.StatsSampleLogger().Warn(
				"Histogram not found, possibly due to DDL event is not handled, please consider analyze the table",
				zap.Int64("tableID", col.TableID),
				zap.Int64("columnID", col.ID),
			)
		}
		return err
	}
	var (
		cms  *statistics.CMSketch
		topN *statistics.TopN
		fms  *statistics.FMSketch
	)
	if fullLoad {
		hg, err = HistogramFromStorageWithPriority(sctx, col.TableID, col.ID, &colInfo.FieldType, hg.NDV, 0, hg.LastUpdateVersion, hg.NullCount, hg.TotColSize, hg.Correlation, kv.PriorityHigh)
		if err != nil {
			return errors.Trace(err)
		}
		cms, topN, err = CMSketchAndTopNFromStorageWithHighPriority(sctx, col.TableID, 0, col.ID, statsVer)
		if err != nil {
			return errors.Trace(err)
		}
		if loadFMSketch {
			fms, err = FMSketchFromStorage(sctx, col.TableID, 0, col.ID)
			if err != nil {
				return errors.Trace(err)
			}
		}
	}

	colHist := &statistics.Column{
		PhysicalID: col.TableID,
		Histogram:  *hg,
		Info:       colInfo,
		CMSketch:   cms,
		TopN:       topN,
		FMSketch:   fms,
		IsHandle:   tblInfo.PKIsHandle && mysql.HasPriKeyFlag(colInfo.GetFlag()),
		StatsVer:   statsVer,
	}
	// Reload the latest stats cache, otherwise the `updateStatsCache` may fail with high probability, because functions
	// like `GetPartitionStats` called in `fmSketchFromStorage` would have modified the stats cache already.
	statsTbl, ok = statsHandle.Get(col.TableID)
	if !ok {
		// This could happen when the table is dropped after the async load is triggered.
		statslogutil.StatsSampleLogger().Info(
			"Table statistics item not found, possibly due to table being dropped",
			zap.Int64("tableID", col.TableID),
			zap.Int64("columnID", col.ID),
		)
		return nil
	}
	statsTbl = statsTbl.Copy()
	if colHist.StatsAvailable() {
		if fullLoad {
			colHist.StatsLoadedStatus = statistics.NewStatsFullLoadStatus()
		} else {
			colHist.StatsLoadedStatus = statistics.NewStatsAllEvictedStatus()
		}
		if statsVer != statistics.Version0 {
			statsTbl.LastAnalyzeVersion = max(statsTbl.LastAnalyzeVersion, colHist.LastUpdateVersion)
			statsTbl.StatsVer = int(statsVer)
		}
	}
	statsTbl.SetCol(col.ID, colHist)
	statsHandle.UpdateStatsCache(statstypes.CacheUpdate{
		Updated: []*statistics.Table{statsTbl},
	})
	if col.IsSyncLoadFailed {
		statslogutil.StatsLogger().Warn("Column histogram loaded asynchronously after sync load failure",
			zap.Int64("tableID", colHist.PhysicalID),
			zap.Int64("columnID", colHist.Info.ID),
			zap.String("columnName", colHist.Info.Name.O))
	}
	return nil
}

// loadNeededIndexHistograms loads the necessary index histograms.
// It is similar to loadNeededColumnHistograms, but for index.
func loadNeededIndexHistograms(sctx sessionctx.Context, is infoschema.InfoSchema, statsHandle statstypes.StatsHandle, idx model.TableItemID, loadFMSketch bool) (err error) {
	// Regardless of whether the load is successful or not, we must remove the item from the async load list.
	// The principle is to load the histogram for each index at most once in async load, as we already have a retry mechanism in the sync load.
	defer asyncload.AsyncLoadHistogramNeededItems.Delete(idx)

	tbl, ok := statsHandle.Get(idx.TableID)
	if !ok {
		// This could happen when the table is dropped after the async load is triggered.
		statslogutil.StatsSampleLogger().Info(
			"Table statistics item not found, possibly due to table being dropped",
			zap.Int64("tableID", idx.TableID),
			zap.Int64("indexID", idx.ID),
		)
		return nil
	}
	_, loadNeeded := tbl.IndexIsLoadNeeded(idx.ID)
	if !loadNeeded {
		return nil
	}
	hgMeta, statsVer, err := HistMetaFromStorageWithHighPriority(sctx, &idx, nil)
	if hgMeta == nil || err != nil {
		if hgMeta == nil {
			statslogutil.StatsLogger().Warn(
				"Histogram not found, possibly due to DDL event is not handled, please consider analyze the table",
				zap.Int64("tableID", idx.TableID),
				zap.Int64("indexID", idx.ID),
			)
		}
		return err
	}
	tblInfo, ok := statsHandle.TableInfoByID(is, idx.TableID)
	if !ok {
		// This could happen when the table is dropped after the async load is triggered.
		statslogutil.StatsSampleLogger().Info(
			"Table information not found, possibly due to table being dropped",
			zap.Int64("tableID", idx.TableID),
			zap.Int64("indexID", idx.ID),
		)
		return nil
	}
	idxInfo := tblInfo.Meta().FindIndexByID(idx.ID)
	if idxInfo == nil {
		// This could happen when the index is dropped after the async load is triggered.
		statslogutil.StatsSampleLogger().Info(
			"Index information not found, possibly due to index being dropped",
			zap.Int64("tableID", idx.TableID),
			zap.Int64("indexID", idx.ID),
		)
		return nil
	}
	hg, err := HistogramFromStorageWithPriority(sctx, idx.TableID, idx.ID, types.NewFieldType(mysql.TypeBlob), hgMeta.NDV, 1, hgMeta.LastUpdateVersion, hgMeta.NullCount, hgMeta.TotColSize, hgMeta.Correlation, kv.PriorityHigh)
	if err != nil {
		return errors.Trace(err)
	}
	cms, topN, err := CMSketchAndTopNFromStorageWithHighPriority(sctx, idx.TableID, 1, idx.ID, statsVer)
	if err != nil {
		return errors.Trace(err)
	}
	var fms *statistics.FMSketch
	if loadFMSketch {
		fms, err = FMSketchFromStorage(sctx, idx.TableID, 1, idx.ID)
		if err != nil {
			return errors.Trace(err)
		}
	}
	idxHist := &statistics.Index{
		Histogram:         *hg,
		CMSketch:          cms,
		TopN:              topN,
		FMSketch:          fms,
		Info:              idxInfo,
		StatsVer:          statsVer,
		PhysicalID:        idx.TableID,
		StatsLoadedStatus: statistics.NewStatsFullLoadStatus(),
	}

	tbl, ok = statsHandle.Get(idx.TableID)
	if !ok {
		// This could happen when the table is dropped after the async load is triggered.
		statslogutil.StatsSampleLogger().Info(
			"Table statistics item not found, possibly due to table being dropped",
			zap.Int64("tableID", idx.TableID),
			zap.Int64("indexID", idx.ID),
		)
		return nil
	}
	tbl = tbl.Copy()
	if idxHist.StatsVer != statistics.Version0 {
		tbl.StatsVer = int(idxHist.StatsVer)
		tbl.LastAnalyzeVersion = max(tbl.LastAnalyzeVersion, idxHist.LastUpdateVersion)
	}
	tbl.SetIdx(idx.ID, idxHist)
	statsHandle.UpdateStatsCache(statstypes.CacheUpdate{
		Updated: []*statistics.Table{tbl},
	})
	if idx.IsSyncLoadFailed {
		statslogutil.StatsLogger().Warn("Index histogram loaded asynchronously after sync load failure",
			zap.Int64("tableID", idx.TableID),
			zap.Int64("indexID", idxHist.Info.ID),
			zap.String("indexName", idxHist.Info.Name.O))
	}
	return nil
}

// StatsMetaByTableIDFromStorage gets the stats meta of a table from storage.
func StatsMetaByTableIDFromStorage(sctx sessionctx.Context, tableID int64, snapshot uint64) (version uint64, modifyCount, count int64, err error) {
	var rows []chunk.Row
	if snapshot == 0 {
		rows, _, err = util.ExecRows(sctx,
			"SELECT version, modify_count, count from mysql.stats_meta where table_id = %? order by version", tableID)
	} else {
		rows, _, err = util.ExecWithOpts(sctx,
			[]sqlexec.OptionFuncAlias{sqlexec.ExecOptionWithSnapshot(snapshot), sqlexec.ExecOptionUseCurSession},
			"SELECT version, modify_count, count from mysql.stats_meta where table_id = %? order by version", tableID)
	}
	if err != nil || len(rows) == 0 {
		return
	}
	version = rows[0].GetUint64(0)
	modifyCount = rows[0].GetInt64(1)
	count = rows[0].GetInt64(2)
	return
}

// convertBoundFromBlob reads the bound from blob. The `blob` is read from the `mysql.stats_buckets` table.
// The `convertBoundFromBlob(convertBoundToBlob(a))` should be equal to `a`.
// TODO: add a test to make sure that this assumption is correct.
func convertBoundFromBlob(ctx types.Context, blob types.Datum, tp *types.FieldType) (types.Datum, error) {
	// For `BIT` type, when converting to `BLOB`, it's formated as an integer (when it's possible). Therefore, we should try to
	// parse it as an integer first.
	if tp.GetType() == mysql.TypeBit {
		var ret types.Datum

		// The implementation of converting BIT to BLOB will try to format it as an integer first. Theoretically, it should
		// always be able to format the integer because the `BIT` length is limited to 64. Therefore, this err should never
		// happen.
		uintValue, err := strconv.ParseUint(string(blob.GetBytes()), 10, 64)
		intest.AssertNoError(err)
		if err != nil {
			// Fail to parse, return the original blob as BIT directly.
			ret.SetBinaryLiteral(types.BinaryLiteral(blob.GetBytes()))
			return ret, nil
		}

		// part of the code is copied from `(*Datum).convertToMysqlBit`.
		if tp.GetFlen() < 64 && uintValue >= 1<<(uint64(tp.GetFlen())) {
			logutil.BgLogger().Warn("bound in stats exceeds the bit length", zap.Uint64("bound", uintValue), zap.Int("flen", tp.GetFlen()))
			err = types.ErrDataTooLong.GenWithStack("Data Too Long, field len %d", tp.GetFlen())
			intest.Assert(false, "bound in stats exceeds the bit length")
			uintValue = (1 << (uint64(tp.GetFlen()))) - 1
		}
		byteSize := (tp.GetFlen() + 7) >> 3
		ret.SetMysqlBit(types.NewBinaryLiteralFromUint(uintValue, byteSize))
		return ret, errors.Trace(err)
	}
	return blob.ConvertTo(ctx, tp)
}
