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

package hint

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/pingcap/tidb/pkg/domain"
	"github.com/pingcap/tidb/pkg/domain/infosync"
	"github.com/pingcap/tidb/pkg/meta/model"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/sessionctx/variable"
	"github.com/pingcap/tidb/pkg/store/mockstore"
	"github.com/pingcap/tidb/pkg/testkit"
	"github.com/pingcap/tidb/pkg/testkit/testdata"
	"github.com/pingcap/tidb/pkg/testkit/testfailpoint"
	"github.com/stretchr/testify/require"
)

func TestReadFromStorageHint(t *testing.T) {
	store := testkit.CreateMockStore(t)
	tk := testkit.NewTestKit(t, store)

	tk.MustExec("use test")

	tk.MustExec("drop table if exists t, tt, ttt")
	tk.MustExec("set session tidb_allow_mpp=OFF")
	// since allow-mpp is adjusted to false, there will be no physical plan if TiFlash cop is banned.
	tk.MustExec("set @@session.tidb_allow_tiflash_cop=ON")
	tk.MustExec("create table t(a int, b int, index ia(a))")
	tk.MustExec("create table tt(a int, b int, primary key(a))")
	tk.MustExec("create table ttt(a int, primary key (a desc))")

	// Create virtual tiflash replica info.
	dom := domain.GetDomain(tk.Session())
	testkit.SetTiFlashReplica(t, dom, "test", "t")
	testkit.SetTiFlashReplica(t, dom, "test", "tt")
	testkit.SetTiFlashReplica(t, dom, "test", "ttt")

	var input []string
	var output []struct {
		SQL  string
		Plan []string
		Warn []string
	}
	integrationSuiteData := GetIntegrationSuiteData()
	integrationSuiteData.LoadTestCases(t, &input, &output)
	for i, tt := range input {
		testdata.OnRecord(func() {
			output[i].SQL = tt
			output[i].Plan = testdata.ConvertRowsToStrings(tk.MustQuery(tt).Rows())
			output[i].Warn = testdata.ConvertSQLWarnToStrings(tk.Session().GetSessionVars().StmtCtx.GetWarnings())
		})
		res := tk.MustQuery(tt)
		res.Check(testkit.Rows(output[i].Plan...))
		require.Equal(t, output[i].Warn, testdata.ConvertSQLWarnToStrings(tk.Session().GetSessionVars().StmtCtx.GetWarnings()))
	}
}

func TestAllViewHintType(t *testing.T) {
	store := testkit.CreateMockStore(t, mockstore.WithMockTiFlash(2))
	tk := testkit.NewTestKit(t, store)

	tk.MustExec("use test")

	tk.MustExec("set @@session.tidb_allow_mpp=ON")
	tk.MustExec("set @@session.tidb_isolation_read_engines='tiflash, tikv'")
	tk.MustExec("drop view if exists v, v1, v2, v3, v4, v5, v6, v7, v8, v9, v10, v11, v12")
	tk.MustExec("drop table if exists t, t1, t2, t4, t3, t5")
	tk.MustExec("create table t(a int not null, b int, index idx_a(a));")
	tk.MustExec("create table t1(a int not null, b int, index idx_a(a));")
	tk.MustExec("create table t2(a int, b int, index idx_a(a));")
	tk.MustExec("create table t3(a int, b int, index idx_a(a));")
	tk.MustExec("create table t4(a int, b int, index idx_a(a));")
	tk.MustExec("create table t5(a int, b int, index idx_a(a), index idx_b(b));")

	// Create virtual tiflash replica info.
	dom := domain.GetDomain(tk.Session())
	is := dom.InfoSchema()
	tbl, err := is.TableByName(context.Background(), ast.NewCIStr("test"), ast.NewCIStr("t"))
	require.NoError(t, err)
	tbl.Meta().TiFlashReplica = &model.TiFlashReplicaInfo{
		Count:     1,
		Available: true,
	}

	tk.MustExec("create definer='root'@'localhost' view v as select t.a, t.b from t join t1 on t.a = t1.a;")
	tk.MustExec("create definer='root'@'localhost' view v1 as select t2.a, t2.b from t2 join t3 join v where t2.b = t3.b and t3.a = v.a;")
	tk.MustExec("create definer='root'@'localhost' view v2 as select t.a, t.b from t join (select count(*) as a from t1 join v1 on t1.b=v1.b group by v1.a) tt on t.a = tt.a;")
	tk.MustExec("create definer='root'@'localhost' view v3 as select * from t5 where a > 1 and b < 2;")
	tk.MustExec("create definer='root'@'localhost' view v4 as select * from t5 where a > 1 or b < 2;")
	tk.MustExec("create definer='root'@'localhost' view v5 as SELECT * FROM t WHERE EXISTS (SELECT 1 FROM t1 WHERE t1.b = t.b);")
	tk.MustExec("create definer='root'@'localhost' view v6 as select * from t1 where t1.a < (select sum(t2.a) from t2 where t2.b = t1.b);")
	tk.MustExec("create definer='root'@'localhost' view v7 as WITH CTE AS (SELECT * FROM t WHERE t.a < 60) SELECT * FROM CTE WHERE CTE.a <18 union select * from cte where cte.b > 1;")
	tk.MustExec("create definer='root'@'localhost' view v8 as WITH CTE1 AS (SELECT b FROM t1), CTE2 AS (WITH CTE3 AS (SELECT a FROM t2), CTE4 AS (SELECT a FROM t3) SELECT CTE3.a FROM CTE3, CTE4) SELECT b FROM CTE1, CTE2 union select * from CTE1;")
	tk.MustExec("create definer='root'@'localhost' view v9 as select sum(a) from t;")
	tk.MustExec("create definer='root'@'localhost' view v10 as SELECT * FROM t WHERE a > 10 ORDER BY b LIMIT 1;")
	tk.MustExec("create definer='root'@'localhost' view v11 as select a, sum(b) from t group by a")
	tk.MustExec("create definer='root'@'localhost' view v12 as select t.a, t.b from t join t t1 on t.a = t1.a;")

	var input []string
	var output []struct {
		SQL  string
		Plan []string
		Warn []string
	}
	integrationSuiteData := GetIntegrationSuiteData()
	integrationSuiteData.LoadTestCases(t, &input, &output)
	for i, tt := range input {
		testdata.OnRecord(func() {
			output[i].SQL = tt
			output[i].Plan = testdata.ConvertRowsToStrings(tk.MustQuery(tt).Rows())
			output[i].Warn = testdata.ConvertSQLWarnToStrings(tk.Session().GetSessionVars().StmtCtx.GetWarnings())
		})
		res := tk.MustQuery(tt)
		res.Check(testkit.Rows(output[i].Plan...))
		require.Equal(t, output[i].Warn, testdata.ConvertSQLWarnToStrings(tk.Session().GetSessionVars().StmtCtx.GetWarnings()))
	}
}

func TestJoinHintCompatibility(t *testing.T) {
	store := testkit.CreateMockStore(t, mockstore.WithMockTiFlash(2))
	tk := testkit.NewTestKit(t, store)

	tk.MustExec("use test")

	tk.MustExec("set @@session.tidb_allow_mpp=ON")
	tk.MustExec("set @@session.tidb_isolation_read_engines='tiflash, tikv'")
	tk.MustExec("drop view if exists v, v1, v2")
	tk.MustExec("drop table if exists t, t1, t2, t3, t4, t5, t6, t7, t8, t9;")
	tk.MustExec("create table t(a int not null, b int, index idx_a(a), index idx_b(b));")
	tk.MustExec("create table t1(a int not null, b int, index idx_a(a), index idx_b(b));")
	tk.MustExec("create table t2(a int, b int, index idx_a(a), index idx_b(b));")
	tk.MustExec("create table t3(a int, b int, index idx_a(a), index idx_b(b));")
	tk.MustExec("create table t4(a int, b int, index idx_a(a), index idx_b(b));")
	tk.MustExec("create table t5(a int, b int, index idx_a(a), index idx_b(b));")
	tk.MustExec("create table t6(a int, b int, index idx_a(a), index idx_b(b));")
	tk.MustExec("create table t7(a int, b int, index idx_a(a), index idx_b(b)) partition by hash(a) partitions 4;")
	tk.MustExec("create table t8(a int, b int, index idx_a(a), index idx_b(b)) partition by hash(a) partitions 4;")
	tk.MustExec("create table t9(a int, b int, index idx_a(a), index idx_b(b)) partition by hash(a) partitions 4;")
	tk.MustExec("analyze table t7, t8, t9")

	// Create virtual tiflash replica info.
	dom := domain.GetDomain(tk.Session())
	testkit.SetTiFlashReplica(t, dom, "test", "t4")
	testkit.SetTiFlashReplica(t, dom, "test", "t5")
	testkit.SetTiFlashReplica(t, dom, "test", "t6")

	tk.MustExec("create definer='root'@'localhost' view v as select /*+ leading(t1), inl_join(t1) */ t.a from t join t1 join t2 where t.a = t1.a and t1.b = t2.b;")
	tk.MustExec("create definer='root'@'localhost' view v1 as select /*+ leading(t2), merge_join(t) */ t.a from t join t1 join t2 where t.a = t1.a and t1.b = t2.b;")
	tk.MustExec("create definer='root'@'localhost' view v2 as select t.a from t join t1 join t2 where t.a = t1.a and t1.b = t2.b;")

	var input []string
	var output []struct {
		SQL  string
		Plan []string
		Warn []string
	}
	integrationSuiteData := GetIntegrationSuiteData()
	integrationSuiteData.LoadTestCases(t, &input, &output)
	for i, tt := range input {
		testdata.OnRecord(func() {
			output[i].SQL = tt
			output[i].Plan = testdata.ConvertRowsToStrings(tk.MustQuery(tt).Rows())
			output[i].Warn = testdata.ConvertSQLWarnToStrings(tk.Session().GetSessionVars().StmtCtx.GetWarnings())
		})
		res := tk.MustQuery(tt)
		res.Check(testkit.Rows(output[i].Plan...))
		require.Equal(t, output[i].Warn, testdata.ConvertSQLWarnToStrings(tk.Session().GetSessionVars().StmtCtx.GetWarnings()))
	}
}

func TestReadFromStorageHintAndIsolationRead(t *testing.T) {
	store := testkit.CreateMockStore(t)
	tk := testkit.NewTestKit(t, store)

	tk.MustExec("use test")

	tk.MustExec("drop table if exists t, tt, ttt")
	tk.MustExec("create table t(a int, b int, index ia(a))")
	tk.MustExec("set @@session.tidb_isolation_read_engines=\"tikv\"")

	// Create virtual tiflash replica info.
	dom := domain.GetDomain(tk.Session())
	testkit.SetTiFlashReplica(t, dom, "test", "t")

	var input []string
	var output []struct {
		SQL  string
		Plan []string
		Warn []string
	}
	integrationSuiteData := GetIntegrationSuiteData()
	integrationSuiteData.LoadTestCases(t, &input, &output)
	for i, tt := range input {
		tk.Session().GetSessionVars().StmtCtx.SetWarnings(nil)
		testdata.OnRecord(func() {
			output[i].SQL = tt
			output[i].Plan = testdata.ConvertRowsToStrings(tk.MustQuery(tt).Rows())
			output[i].Warn = testdata.ConvertSQLWarnToStrings(tk.Session().GetSessionVars().StmtCtx.GetWarnings())
		})
		res := tk.MustQuery(tt)
		res.Check(testkit.Rows(output[i].Plan...))
		require.Equal(t, output[i].Warn, testdata.ConvertSQLWarnToStrings(tk.Session().GetSessionVars().StmtCtx.GetWarnings()))
	}
}

func TestIsolationReadTiFlashUseIndexHint(t *testing.T) {
	store, dom := testkit.CreateMockStoreAndDomainWithSchemaLease(t, 200*time.Millisecond, mockstore.WithMockTiFlash(2))
	tk := testkit.NewTestKit(t, store)

	tiflash := infosync.NewMockTiFlash()
	infosync.SetMockTiFlash(tiflash)
	defer func() {
		tiflash.Lock()
		tiflash.StatusServer.Close()
		tiflash.Unlock()
	}()

	testfailpoint.Enable(t, "github.com/pingcap/tidb/pkg/ddl/MockCheckColumnarIndexProcess", `return(1)`)

	tk.MustExec("use test")
	tk.MustExec("drop table if exists t")
	tk.MustExec("create table t(a int, vec vector(3), PRIMARY KEY (`a`) /*T![clustered_index] CLUSTERED */, index idx(a), vector index idx_vec ((VEC_COSINE_DISTANCE(`vec`))) USING HNSW);")
	tblInfo, err := dom.InfoSchema().TableByName(context.Background(), ast.NewCIStr("test"), ast.NewCIStr("t"))
	require.NoError(t, err)
	err = domain.GetDomain(tk.Session()).DDLExecutor().UpdateTableReplicaInfo(tk.Session(), tblInfo.Meta().ID, true)
	require.NoError(t, err)

	tk.MustExec("set @@session.tidb_isolation_read_engines=\"tiflash\"")
	var input []string
	var output []struct {
		SQL  string
		Plan []string
		Warn []string
	}
	integrationSuiteData := GetIntegrationSuiteData()
	integrationSuiteData.LoadTestCases(t, &input, &output)
	for i, tt := range input {
		testdata.OnRecord(func() {
			output[i].SQL = tt
			output[i].Plan = testdata.ConvertRowsToStrings(tk.MustQuery(tt).Rows())
			output[i].Warn = testdata.ConvertSQLWarnToStrings(tk.Session().GetSessionVars().StmtCtx.GetWarnings())
		})
		res := tk.MustQuery(tt)
		res.Check(testkit.Rows(output[i].Plan...))
		require.Equal(t, output[i].Warn, testdata.ConvertSQLWarnToStrings(tk.Session().GetSessionVars().StmtCtx.GetWarnings()))
	}
}

func TestOptimizeHintOnPartitionTable(t *testing.T) {
	store := testkit.CreateMockStore(t)
	tk := testkit.NewTestKit(t, store)

	tk.MustExec("use test")
	tk.MustExec("drop table if exists t")
	tk.MustExec(`create table t (
					a int, b int, c varchar(20),
					primary key(a), key(b), key(c)
				) partition by range columns(a) (
					partition p0 values less than(6),
					partition p1 values less than(11),
					partition p2 values less than(16));`)
	tk.MustExec(`insert into t values (1,1,"1"), (2,2,"2"), (8,8,"8"), (11,11,"11"), (15,15,"15")`)
	tk.MustExec("set @@tidb_enable_index_merge = off")
	defer func() {
		tk.MustExec("set @@tidb_enable_index_merge = on")
	}()

	// Create virtual tiflash replica info.
	dom := domain.GetDomain(tk.Session())
	is := dom.InfoSchema()
	tbl, err := is.TableByName(context.Background(), ast.NewCIStr("test"), ast.NewCIStr("t"))
	require.NoError(t, err)
	tbl.Meta().TiFlashReplica = &model.TiFlashReplicaInfo{
		Count:     1,
		Available: true,
	}

	tk.MustExec(`set @@tidb_partition_prune_mode='` + string(variable.Static) + `'`)

	var input []string
	var output []struct {
		SQL  string
		Plan []string
		Warn []string
	}
	integrationSuiteData := GetIntegrationSuiteData()
	integrationSuiteData.LoadTestCases(t, &input, &output)
	for i, tt := range input {
		testdata.OnRecord(func() {
			output[i].SQL = tt
			output[i].Plan = testdata.ConvertRowsToStrings(tk.MustQuery("explain format = 'brief' " + tt).Rows())
			output[i].Warn = testdata.ConvertRowsToStrings(tk.MustQuery("show warnings").Rows())
		})
		tk.MustQuery("explain format = 'brief' " + tt).Check(testkit.Rows(output[i].Plan...))
		tk.MustQuery("show warnings").Check(testkit.Rows(output[i].Warn...))
	}
	tk.MustQuery("SELECT /*+ MAX_EXECUTION_TIME(10)  */ SLEEP(5)").Check(testkit.Rows("0"))
	tk.MustQuery("SELECT /*+ MAX_EXECUTION_TIME(10), dtc(name=tt)  */ SLEEP(5)").Check(testkit.Rows("0"))
	require.Len(t, tk.Session().GetSessionVars().StmtCtx.GetWarnings(), 1)
	tk.MustQuery("SELECT /*+ MAX_EXECUTION_TIME(10), dtc(name=tt) unknow(t1,t2) */ SLEEP(5)").Check(testkit.Rows("0"))
	require.Len(t, tk.Session().GetSessionVars().StmtCtx.GetWarnings(), 2)
}

func TestHints(t *testing.T) {
	store := testkit.CreateMockStore(t)
	tk := testkit.NewTestKit(t, store)
	tk.MustExec("use test")
	tk.MustExec("create table t1 (a int);")
	tk.MustExec("create table t2 (a int);")
	tk.MustExec("create table t3 (a int);")
	var input []string
	var output []struct {
		SQL  string
		Plan []string
		Warn []string
	}
	integrationSuiteData := GetIntegrationSuiteData()
	integrationSuiteData.LoadTestCases(t, &input, &output)
	for i, tt := range input {
		testdata.OnRecord(func() {
			output[i].SQL = tt
			output[i].Plan = testdata.ConvertRowsToStrings(tk.MustQuery("explain format = 'brief' " + tt).Rows())
			output[i].Warn = testdata.ConvertRowsToStrings(tk.MustQuery("show warnings").Rows())
		})
		tk.MustQuery("explain format = 'brief' " + tt).Check(testkit.Rows(output[i].Plan...))
		tk.MustQuery("show warnings").Check(testkit.Rows(output[i].Warn...))
	}
}

func TestQBHintHandlerDuplicateObjects(t *testing.T) {
	store := testkit.CreateMockStore(t)
	tk := testkit.NewTestKit(t, store)
	tk.MustExec("use test")
	tk.MustExec("CREATE TABLE t_employees  (id INT NOT NULL AUTO_INCREMENT PRIMARY KEY, fname VARCHAR(25) NOT NULL, lname VARCHAR(25) NOT NULL, store_id INT NOT NULL, department_id INT NOT NULL);")
	tk.MustExec("ALTER TABLE t_employees ADD INDEX idx(department_id);")

	// Explain statement
	tk.MustQuery("EXPLAIN WITH t AS (SELECT /*+ inl_join(e) */ em.* FROM t_employees em JOIN t_employees e WHERE em.store_id = e.department_id) SELECT * FROM t;")
	tk.MustQuery("show warnings").Check(testkit.Rows())
}

func TestOptimizerCostFactorHints(t *testing.T) {
	// This test covers a subset of the cost factorst - using hints to increase the cost factor
	// of a specific plan type. The test is not exhaustive and does not cover all cost factors.
	store := testkit.CreateMockStore(t)

	tk := testkit.NewTestKit(t, store)
	tk.MustExec("use test")
	tk.MustExec("drop table if exists t")
	tk.MustExec("create table t(a int, b int, c int, primary key (a), key(b))")
	// Insert some data
	tk.MustExec("insert into t values (1,1,1),(2,2,2),(3,3,3),(4,4,4),(5,5,5)")
	// Analyze table to update statistics
	tk.MustExec("analyze table t")

	// Test tableFullScan cost factor increase via hint
	// Set index scan cost factor variable to isolate testing to TableFullScan
	tk.MustExec("set @@session.tidb_opt_index_scan_cost_factor=100")
	rs := tk.MustQuery("explain format=verbose select * from t").Rows()
	planCost1, err1 := strconv.ParseFloat(rs[0][2].(string), 64)
	require.Nil(t, err1)
	rs = tk.MustQuery("explain format=verbose select /*+ SET_VAR(tidb_opt_table_full_scan_cost_factor=2) */ * from t").Rows()
	planCost2, err2 := strconv.ParseFloat(rs[0][2].(string), 64)
	require.Nil(t, err2)
	// 1st query should be cheaper than 2nd query
	require.Less(t, planCost1, planCost2)
	// Reset to index scan to default
	tk.MustExec("set @@session.tidb_opt_index_scan_cost_factor=1")

	// Test tableReader cost factor increase via hint
	// Set index scan cost factor variable to isolate testing to TableReader
	tk.MustExec("set @@session.tidb_opt_index_scan_cost_factor=100")
	rs = tk.MustQuery("explain format=verbose select * from t").Rows()
	planCost1, err1 = strconv.ParseFloat(rs[0][2].(string), 64)
	require.Nil(t, err1)
	rs = tk.MustQuery("explain format=verbose select /*+ SET_VAR(tidb_opt_table_reader_cost_factor=2) */ * from t").Rows()
	planCost2, err2 = strconv.ParseFloat(rs[0][2].(string), 64)
	require.Nil(t, err2)
	// 1st query should be cheaper than 2nd query
	require.Less(t, planCost1, planCost2)
	// Reset to index scan variable to default
	tk.MustExec("set @@session.tidb_opt_index_scan_cost_factor=1")

	// Test tableRangeScan cost factor increase
	// Set index scan and table full scan cost factor variables to isolate testing to TableRangeScan
	tk.MustExec("set @@session.tidb_opt_index_scan_cost_factor=100")
	tk.MustExec("set @@session.tidb_opt_table_full_scan_cost_factor=100")
	rs = tk.MustQuery("explain format=verbose select * from t where a > 3").Rows()
	planCost1, err1 = strconv.ParseFloat(rs[0][2].(string), 64)
	require.Nil(t, err1)
	rs = tk.MustQuery("explain format=verbose select /*+ SET_VAR(tidb_opt_table_range_scan_cost_factor=2) */ * from t where a > 3").Rows()
	planCost2, err2 = strconv.ParseFloat(rs[0][2].(string), 64)
	require.Nil(t, err2)
	// 1st query should be cheaper than 2nd query
	require.Less(t, planCost1, planCost2)
	// Reset to index and table full scan variables to default
	tk.MustExec("set @@session.tidb_opt_index_scan_cost_factor=1")
	tk.MustExec("set @@session.tidb_opt_table_full_scan_cost_factor=1")

	// Test IndexScan cost factor increase
	// Increase table scan cost factor to isolate testing to IndexScan
	tk.MustExec("set @@session.tidb_opt_table_full_scan_cost_factor=100")
	rs = tk.MustQuery("explain format=verbose select b from t where b > 3").Rows()
	planCost1, err1 = strconv.ParseFloat(rs[0][2].(string), 64)
	require.Nil(t, err1)
	rs = tk.MustQuery("explain format=verbose select /*+ SET_VAR(tidb_opt_index_scan_cost_factor=2) */ b from t where b > 3").Rows()
	planCost2, err2 = strconv.ParseFloat(rs[0][2].(string), 64)
	require.Nil(t, err2)
	// 1st query should be cheaper than 2nd query
	require.Less(t, planCost1, planCost2)
	// Reset table full scan variable to default
	tk.MustExec("set @@session.tidb_opt_table_full_scan_cost_factor=1")

	// Test IndexReadercost factor increase
	// Increase table scan cost factor variable to isolate testing to IndexReader
	tk.MustExec("set @@session.tidb_opt_table_full_scan_cost_factor=100")
	rs = tk.MustQuery("explain format=verbose select b from t where b > 3").Rows()
	planCost1, err1 = strconv.ParseFloat(rs[0][2].(string), 64)
	require.Nil(t, err1)
	rs = tk.MustQuery("explain format=verbose select  /*+ SET_VAR(tidb_opt_index_reader_cost_factor=2) */ b from t where b > 3").Rows()
	planCost2, err2 = strconv.ParseFloat(rs[0][2].(string), 64)
	require.Nil(t, err2)
	// 1st query should be cheaper than 2nd query
	require.Less(t, planCost1, planCost2)
	// Reset table scan cost factor variable to default
	tk.MustExec("set @@session.tidb_opt_table_full_scan_cost_factor=1")
}
