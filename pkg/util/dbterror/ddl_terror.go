// Copyright 2022 PingCAP, Inc.
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

package dbterror

import (
	"fmt"

	mysql "github.com/pingcap/tidb/pkg/errno"
	parser_mysql "github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pingcap/tidb/pkg/parser/terror"
)

var (
	// ErrInvalidWorker means the worker is invalid.
	ErrInvalidWorker = ClassDDL.NewStd(mysql.ErrInvalidDDLWorker)
	// ErrNotOwner means we are not owner and can't handle DDL jobs.
	ErrNotOwner = ClassDDL.NewStd(mysql.ErrNotOwner)
	// ErrCantDecodeRecord means we can't decode the record.
	ErrCantDecodeRecord = ClassDDL.NewStd(mysql.ErrCantDecodeRecord)
	// ErrInvalidDDLJob means the DDL job is invalid.
	ErrInvalidDDLJob = ClassDDL.NewStd(mysql.ErrInvalidDDLJob)
	// ErrCancelledDDLJob means the DDL job is cancelled.
	ErrCancelledDDLJob = ClassDDL.NewStd(mysql.ErrCancelledDDLJob)
	// ErrPausedDDLJob returns when the DDL job cannot be paused.
	ErrPausedDDLJob = ClassDDL.NewStd(mysql.ErrPausedDDLJob)
	// ErrBDRRestrictedDDL means the DDL is restricted in BDR mode.
	ErrBDRRestrictedDDL = ClassDDL.NewStd(mysql.ErrBDRRestrictedDDL)
	// ErrRunMultiSchemaChanges means we run multi schema changes.
	ErrRunMultiSchemaChanges = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "multi schema change for %s"), nil))
	// ErrOperateSameColumn means we change the same columns multiple times in a DDL.
	ErrOperateSameColumn = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "operate same column '%s'"), nil))
	// ErrOperateSameIndex means we change the same indexes multiple times in a DDL.
	ErrOperateSameIndex = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "operate same index '%s'"), nil))
	// ErrWaitReorgTimeout means we wait for reorganization timeout.
	ErrWaitReorgTimeout = ClassDDL.NewStdErr(mysql.ErrLockWaitTimeout, mysql.MySQLErrName[mysql.ErrWaitReorgTimeout])
	// ErrInvalidStoreVer means invalid store version.
	ErrInvalidStoreVer = ClassDDL.NewStd(mysql.ErrInvalidStoreVersion)
	// ErrRepairTableFail is used to repair tableInfo in repair mode.
	ErrRepairTableFail = ClassDDL.NewStd(mysql.ErrRepairTable)

	// ErrUnsupportedAddColumnarIndex means add columnar index is unsupported
	ErrUnsupportedAddColumnarIndex = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "add columnar index: %s"), nil))
	// ErrUnsupportedAddVectorIndex means add vector index is unsupported
	ErrUnsupportedAddVectorIndex = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "add vector index: %s"), nil))
	// ErrCantDropColWithIndex means can't drop the column with index. We don't support dropping column with index covered now.
	ErrCantDropColWithIndex = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "drop column with index"), nil))
	// ErrCantDropColWithAutoInc means can't drop column with auto_increment
	ErrCantDropColWithAutoInc = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "can't remove column with auto_increment when @@tidb_allow_remove_auto_inc disabled"), nil))
	// ErrCantDropColWithCheckConstraint means can't drop column with check constraint
	ErrCantDropColWithCheckConstraint = ClassDDL.NewStd(mysql.ErrDependentByCheckConstraint)
	// ErrUnsupportedEngineAttribute means engine attribute option is unsupported
	ErrUnsupportedEngineAttribute = ClassDDL.NewStd(mysql.ErrEngineAttributeNotSupported)
	// ErrUnsupportedAddColumn means add columns is unsupported
	ErrUnsupportedAddColumn = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "add column"), nil))
	// ErrUnsupportedModifyColumn means modify columns is unsupoorted
	ErrUnsupportedModifyColumn = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "modify column: %s"), nil))
	// ErrUnsupportedModifyCharset means modify charset is unsupoorted
	ErrUnsupportedModifyCharset = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "modify %s"), nil))
	// ErrUnsupportedModifyCollation means modify collation is unsupoorted
	ErrUnsupportedModifyCollation = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "modifying collation from %s to %s"), nil))
	// ErrUnsupportedPKHandle is used to indicate that we can't support this PK handle.
	ErrUnsupportedPKHandle = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "drop integer primary key"), nil))
	// ErrUnsupportedCharset means we don't support the charset.
	ErrUnsupportedCharset = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "charset %s and collate %s"), nil))
	// ErrUnsupportedShardRowIDBits means we don't support the shard_row_id_bits.
	ErrUnsupportedShardRowIDBits = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "shard_row_id_bits for table with primary key as row id"), nil))
	// ErrUnsupportedAlterTableWithValidation means we don't support the alter table with validation.
	ErrUnsupportedAlterTableWithValidation = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message("ALTER TABLE WITH VALIDATION is currently unsupported", nil))
	// ErrUnsupportedAlterTableWithoutValidation means we don't support the alter table without validation.
	ErrUnsupportedAlterTableWithoutValidation = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message("ALTER TABLE WITHOUT VALIDATION is currently unsupported", nil))
	// ErrUnsupportedAlterTableOption means we don't support the alter table option.
	ErrUnsupportedAlterTableOption = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message("This type of ALTER TABLE is currently unsupported", nil))
	// ErrUnsupportedAlterCacheForSysTable means we don't support the alter cache for system table.
	ErrUnsupportedAlterCacheForSysTable = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message("ALTER table cache for tables in system database is currently unsupported", nil))
	// ErrBlobKeyWithoutLength is used when BLOB is used as key but without a length.
	ErrBlobKeyWithoutLength = ClassDDL.NewStd(mysql.ErrBlobKeyWithoutLength)
	// ErrKeyPart0 is used when key parts length is 0.
	ErrKeyPart0 = ClassDDL.NewStd(mysql.ErrKeyPart0)
	// ErrIncorrectPrefixKey is used when the prefix length is incorrect for a string key.
	ErrIncorrectPrefixKey = ClassDDL.NewStd(mysql.ErrWrongSubKey)
	// ErrTooLongKey is used when the column key is too long.
	ErrTooLongKey = ClassDDL.NewStd(mysql.ErrTooLongKey)
	// ErrKeyColumnDoesNotExits is used when the key column doesn't exist.
	ErrKeyColumnDoesNotExits = ClassDDL.NewStd(mysql.ErrKeyColumnDoesNotExits)
	// ErrInvalidDDLJobVersion is used when the DDL job version is invalid.
	ErrInvalidDDLJobVersion = ClassDDL.NewStd(mysql.ErrInvalidDDLJobVersion)
	// ErrInvalidUseOfNull is used when the column is not null.
	ErrInvalidUseOfNull = ClassDDL.NewStd(mysql.ErrInvalidUseOfNull)
	// ErrTooManyFields is used when too many columns are used in a select statement.
	ErrTooManyFields = ClassDDL.NewStd(mysql.ErrTooManyFields)
	// ErrTooManyKeys is used when too many keys used.
	ErrTooManyKeys = ClassDDL.NewStd(mysql.ErrTooManyKeys)
	// ErrInvalidSplitRegionRanges is used when split region ranges is invalid.
	ErrInvalidSplitRegionRanges = ClassDDL.NewStd(mysql.ErrInvalidSplitRegionRanges)
	// ErrReorgPanic is used when reorg process is panic.
	ErrReorgPanic = ClassDDL.NewStd(mysql.ErrReorgPanic)
	// ErrFkColumnCannotDrop is used when foreign key column can't be dropped.
	ErrFkColumnCannotDrop = ClassDDL.NewStd(mysql.ErrFkColumnCannotDrop)
	// ErrFkColumnCannotDropChild is used when foreign key column can't be dropped.
	ErrFkColumnCannotDropChild = ClassDDL.NewStd(mysql.ErrFkColumnCannotDropChild)
	// ErrFKIncompatibleColumns is used when foreign key column type is incompatible.
	ErrFKIncompatibleColumns = ClassDDL.NewStd(mysql.ErrFKIncompatibleColumns)
	// ErrOnlyOnRangeListPartition is used when the partition type is range list.
	ErrOnlyOnRangeListPartition = ClassDDL.NewStd(mysql.ErrOnlyOnRangeListPartition)
	// ErrWrongKeyColumn is for table column cannot be indexed.
	ErrWrongKeyColumn = ClassDDL.NewStd(mysql.ErrWrongKeyColumn)
	// ErrWrongKeyColumnFunctionalIndex is for expression cannot be indexed.
	ErrWrongKeyColumnFunctionalIndex = ClassDDL.NewStd(mysql.ErrWrongKeyColumnFunctionalIndex)
	// ErrWrongFKOptionForGeneratedColumn is for wrong foreign key reference option on generated columns.
	ErrWrongFKOptionForGeneratedColumn = ClassDDL.NewStd(mysql.ErrWrongFKOptionForGeneratedColumn)
	// ErrUnsupportedOnGeneratedColumn is for unsupported actions on generated columns.
	ErrUnsupportedOnGeneratedColumn = ClassDDL.NewStd(mysql.ErrUnsupportedOnGeneratedColumn)
	// ErrGeneratedColumnNonPrior forbids to refer generated column non prior to it.
	ErrGeneratedColumnNonPrior = ClassDDL.NewStd(mysql.ErrGeneratedColumnNonPrior)
	// ErrDependentByGeneratedColumn forbids to delete columns which are dependent by generated columns.
	ErrDependentByGeneratedColumn = ClassDDL.NewStd(mysql.ErrDependentByGeneratedColumn)
	// ErrJSONUsedAsKey forbids to use JSON as key or index.
	ErrJSONUsedAsKey = ClassDDL.NewStd(mysql.ErrJSONUsedAsKey)
	// ErrBlobCantHaveDefault forbids to give not null default value to TEXT/BLOB/JSON.
	ErrBlobCantHaveDefault = ClassDDL.NewStd(mysql.ErrBlobCantHaveDefault)
	// ErrTooLongIndexComment means the comment for index is too long.
	ErrTooLongIndexComment = ClassDDL.NewStd(mysql.ErrTooLongIndexComment)
	// ErrTooLongTableComment means the comment for table is too long.
	ErrTooLongTableComment = ClassDDL.NewStd(mysql.ErrTooLongTableComment)
	// ErrTooLongFieldComment means the comment for field/column is too long.
	ErrTooLongFieldComment = ClassDDL.NewStd(mysql.ErrTooLongFieldComment)
	// ErrTooLongTablePartitionComment means the comment for table partition is too long.
	ErrTooLongTablePartitionComment = ClassDDL.NewStd(mysql.ErrTooLongTablePartitionComment)
	// ErrInvalidDefaultValue returns for invalid default value for columns.
	ErrInvalidDefaultValue = ClassDDL.NewStd(mysql.ErrInvalidDefault)
	// ErrDefValGeneratedNamedFunctionIsNotAllowed returns for disallowed function as default value expression of column.
	ErrDefValGeneratedNamedFunctionIsNotAllowed = ClassDDL.NewStd(mysql.ErrDefValGeneratedNamedFunctionIsNotAllowed)
	// ErrGeneratedColumnRefAutoInc forbids to refer generated columns to auto-increment columns .
	ErrGeneratedColumnRefAutoInc = ClassDDL.NewStd(mysql.ErrGeneratedColumnRefAutoInc)
	// ErrExpressionIndexCanNotRefer forbids to refer expression index to auto-increment column.
	ErrExpressionIndexCanNotRefer = ClassDDL.NewStd(mysql.ErrFunctionalIndexRefAutoIncrement)
	// ErrUnsupportedAddPartition returns for does not support add partitions.
	ErrUnsupportedAddPartition = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "add partitions"), nil))
	// ErrUnsupportedCoalescePartition returns for does not support coalesce partitions.
	ErrUnsupportedCoalescePartition = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "coalesce partitions"), nil))
	// ErrUnsupportedReorganizePartition returns for does not support reorganize partitions.
	ErrUnsupportedReorganizePartition = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "reorganize partition"), nil))
	// ErrUnsupportedCheckPartition returns for does not support check partitions.
	ErrUnsupportedCheckPartition = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "check partition"), nil))
	// ErrUnsupportedOptimizePartition returns for does not support optimize partitions.
	ErrUnsupportedOptimizePartition = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "optimize partition"), nil))
	// ErrUnsupportedRebuildPartition returns for does not support rebuild partitions.
	ErrUnsupportedRebuildPartition = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "rebuild partition"), nil))
	// ErrUnsupportedRemovePartition returns for does not support remove partitions.
	ErrUnsupportedRemovePartition = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "remove partitioning"), nil))
	// ErrUnsupportedRepairPartition returns for does not support repair partitions.
	ErrUnsupportedRepairPartition = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "repair partition"), nil))
	// ErrGeneratedColumnFunctionIsNotAllowed returns for unsupported functions for generated columns.
	ErrGeneratedColumnFunctionIsNotAllowed = ClassDDL.NewStd(mysql.ErrGeneratedColumnFunctionIsNotAllowed)
	// ErrGeneratedColumnRowValueIsNotAllowed returns for generated columns referring to row values.
	ErrGeneratedColumnRowValueIsNotAllowed = ClassDDL.NewStd(mysql.ErrGeneratedColumnRowValueIsNotAllowed)
	// ErrUnsupportedPartitionByRangeColumns returns for does unsupported partition by range columns.
	ErrUnsupportedPartitionByRangeColumns = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "partition by range columns"), nil))
	// ErrFunctionalIndexFunctionIsNotAllowed returns for unsupported functions for functional index.
	ErrFunctionalIndexFunctionIsNotAllowed = ClassDDL.NewStd(mysql.ErrFunctionalIndexFunctionIsNotAllowed)
	// ErrFunctionalIndexRowValueIsNotAllowed returns for functional index referring to row values.
	ErrFunctionalIndexRowValueIsNotAllowed = ClassDDL.NewStd(mysql.ErrFunctionalIndexRowValueIsNotAllowed)
	// ErrUnsupportedCreatePartition returns for does not support create partitions.
	ErrUnsupportedCreatePartition = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "partition type, treat as normal table"), nil))
	// ErrUnsupportedIndexType returns for unsupported index type.
	ErrUnsupportedIndexType = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "index type"), nil))
	// ErrWindowInvalidWindowFuncUse returns for invalid window function use.
	ErrWindowInvalidWindowFuncUse = ClassDDL.NewStd(mysql.ErrWindowInvalidWindowFuncUse)

	// ErrDupKeyName returns for duplicated key name.
	ErrDupKeyName = ClassDDL.NewStd(mysql.ErrDupKeyName)
	// ErrFkDupName returns for duplicated FK name.
	ErrFkDupName = ClassDDL.NewStd(mysql.ErrFkDupName)
	// ErrInvalidDDLState returns for invalid ddl model object state.
	ErrInvalidDDLState = ClassDDL.NewStdErr(mysql.ErrInvalidDDLState, parser_mysql.Message(mysql.MySQLErrName[mysql.ErrInvalidDDLState].Raw, nil))
	// ErrUnsupportedModifyPrimaryKey returns an error when add or drop the primary key.
	// It's exported for testing.
	ErrUnsupportedModifyPrimaryKey = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "%s primary key"), nil))
	// ErrPKIndexCantBeInvisible return an error when primary key is invisible index
	ErrPKIndexCantBeInvisible = ClassDDL.NewStd(mysql.ErrPKIndexCantBeInvisible)

	// ErrColumnBadNull returns for a bad null value.
	ErrColumnBadNull = ClassDDL.NewStd(mysql.ErrBadNull)
	// ErrBadField forbids to refer to unknown column.
	ErrBadField = ClassDDL.NewStd(mysql.ErrBadField)
	// ErrCantRemoveAllFields returns for deleting all columns.
	ErrCantRemoveAllFields = ClassDDL.NewStd(mysql.ErrCantRemoveAllFields)
	// ErrCantDropFieldOrKey returns for dropping a non-existent field or key.
	ErrCantDropFieldOrKey = ClassDDL.NewStd(mysql.ErrCantDropFieldOrKey)
	// ErrInvalidOnUpdate returns for invalid ON UPDATE clause.
	ErrInvalidOnUpdate = ClassDDL.NewStd(mysql.ErrInvalidOnUpdate)
	// ErrTooLongIdent returns for too long name of database/table/column/index.
	ErrTooLongIdent = ClassDDL.NewStd(mysql.ErrTooLongIdent)
	// ErrWrongDBName returns for wrong database name.
	ErrWrongDBName = ClassDDL.NewStd(mysql.ErrWrongDBName)
	// ErrWrongTableName returns for wrong table name.
	ErrWrongTableName = ClassDDL.NewStd(mysql.ErrWrongTableName)
	// ErrWrongColumnName returns for wrong column name.
	ErrWrongColumnName = ClassDDL.NewStd(mysql.ErrWrongColumnName)
	// ErrWrongPartitionName returns for wrong partition name.
	ErrWrongPartitionName = ClassDDL.NewStd(mysql.ErrWrongPartitionName)
	// ErrWrongUsage returns for wrong ddl syntax usage.
	ErrWrongUsage = ClassDDL.NewStd(mysql.ErrWrongUsage)
	// ErrInvalidGroupFuncUse returns for using invalid group functions.
	ErrInvalidGroupFuncUse = ClassDDL.NewStd(mysql.ErrInvalidGroupFuncUse)
	// ErrTableMustHaveColumns returns for missing column when creating a table.
	ErrTableMustHaveColumns = ClassDDL.NewStd(mysql.ErrTableMustHaveColumns)
	// ErrWrongNameForIndex returns for wrong index name.
	ErrWrongNameForIndex = ClassDDL.NewStd(mysql.ErrWrongNameForIndex)
	// ErrUnknownCharacterSet returns unknown character set.
	ErrUnknownCharacterSet = ClassDDL.NewStd(mysql.ErrUnknownCharacterSet)
	// ErrUnknownCollation returns unknown collation.
	ErrUnknownCollation = ClassDDL.NewStd(mysql.ErrUnknownCollation)
	// ErrCollationCharsetMismatch returns when collation not match the charset.
	ErrCollationCharsetMismatch = ClassDDL.NewStd(mysql.ErrCollationCharsetMismatch)
	// ErrConflictingDeclarations return conflict declarations.
	ErrConflictingDeclarations = ClassDDL.NewStdErr(mysql.ErrConflictingDeclarations, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrConflictingDeclarations].Raw, "CHARACTER SET ", "%s", "CHARACTER SET ", "%s"), nil))
	// ErrPrimaryCantHaveNull returns All parts of a PRIMARY KEY must be NOT NULL; if you need NULL in a key, use UNIQUE instead
	ErrPrimaryCantHaveNull = ClassDDL.NewStd(mysql.ErrPrimaryCantHaveNull)
	// ErrErrorOnRename returns error for wrong database name in alter table rename
	ErrErrorOnRename = ClassDDL.NewStd(mysql.ErrErrorOnRename)
	// ErrViewSelectClause returns error for create view with select into clause
	ErrViewSelectClause = ClassDDL.NewStd(mysql.ErrViewSelectClause)
	// ErrViewSelectVariable returns error for create view with select into clause
	ErrViewSelectVariable = ClassDDL.NewStd(mysql.ErrViewSelectVariable)

	// ErrNotAllowedTypeInPartition returns not allowed type error when creating table partition with unsupported expression type.
	ErrNotAllowedTypeInPartition = ClassDDL.NewStd(mysql.ErrFieldTypeNotAllowedAsPartitionField)
	// ErrPartitionMgmtOnNonpartitioned returns it's not a partition table.
	ErrPartitionMgmtOnNonpartitioned = ClassDDL.NewStd(mysql.ErrPartitionMgmtOnNonpartitioned)
	// ErrDropPartitionNonExistent returns error in list of partition.
	ErrDropPartitionNonExistent = ClassDDL.NewStd(mysql.ErrDropPartitionNonExistent)
	// ErrSameNamePartition returns duplicate partition name.
	ErrSameNamePartition = ClassDDL.NewStd(mysql.ErrSameNamePartition)
	// ErrSameNamePartitionField returns duplicate partition field.
	ErrSameNamePartitionField = ClassDDL.NewStd(mysql.ErrSameNamePartitionField)
	// ErrRangeNotIncreasing returns values less than value must be strictly increasing for each partition.
	ErrRangeNotIncreasing = ClassDDL.NewStd(mysql.ErrRangeNotIncreasing)
	// ErrPartitionMaxvalue returns maxvalue can only be used in last partition definition.
	ErrPartitionMaxvalue = ClassDDL.NewStd(mysql.ErrPartitionMaxvalue)
	// ErrMaxvalueInValuesIn returns maxvalue cannot be used in values in.
	ErrMaxvalueInValuesIn = ClassDDL.NewStd(mysql.ErrMaxvalueInValuesIn)
	// ErrDropLastPartition returns cannot remove all partitions, use drop table instead.
	ErrDropLastPartition = ClassDDL.NewStd(mysql.ErrDropLastPartition)
	// ErrTooManyPartitions returns too many partitions were defined.
	ErrTooManyPartitions = ClassDDL.NewStd(mysql.ErrTooManyPartitions)
	// ErrPartitionConstDomain returns partition constant is out of partition function domain.
	ErrPartitionConstDomain = ClassDDL.NewStd(mysql.ErrPartitionConstDomain)
	// ErrPartitionFunctionIsNotAllowed returns this partition function is not allowed.
	ErrPartitionFunctionIsNotAllowed = ClassDDL.NewStd(mysql.ErrPartitionFunctionIsNotAllowed)
	// ErrPartitionFuncNotAllowed returns partition function returns the wrong type.
	ErrPartitionFuncNotAllowed = ClassDDL.NewStd(mysql.ErrPartitionFuncNotAllowed)
	// ErrUniqueKeyNeedAllFieldsInPf returns must include all columns in the table's partitioning function.
	ErrUniqueKeyNeedAllFieldsInPf = ClassDDL.NewStd(mysql.ErrUniqueKeyNeedAllFieldsInPf)
	// ErrWrongExprInPartitionFunc Constant, random or timezone-dependent expressions in (sub)partitioning function are not allowed.
	ErrWrongExprInPartitionFunc = ClassDDL.NewStd(mysql.ErrWrongExprInPartitionFunc)
	// ErrWarnDataTruncated returns data truncated error.
	ErrWarnDataTruncated = ClassDDL.NewStd(mysql.WarnDataTruncated)
	// ErrCoalesceOnlyOnHashPartition returns coalesce partition can only be used on hash/key partitions.
	ErrCoalesceOnlyOnHashPartition = ClassDDL.NewStd(mysql.ErrCoalesceOnlyOnHashPartition)
	// ErrViewWrongList returns create view must include all columns in the select clause
	ErrViewWrongList = ClassDDL.NewStd(mysql.ErrViewWrongList)
	// ErrAlterOperationNotSupported returns when alter operations is not supported.
	ErrAlterOperationNotSupported = ClassDDL.NewStd(mysql.ErrAlterOperationNotSupportedReason)
	// ErrWrongObject returns for wrong object.
	ErrWrongObject = ClassDDL.NewStd(mysql.ErrWrongObject)
	// ErrTableCantHandleFt returns FULLTEXT keys are not supported by table type
	ErrTableCantHandleFt = ClassDDL.NewStd(mysql.ErrTableCantHandleFt)
	// ErrFieldNotFoundPart returns an error when 'partition by columns' are not found in table columns.
	ErrFieldNotFoundPart = ClassDDL.NewStd(mysql.ErrFieldNotFoundPart)
	// ErrWrongTypeColumnValue returns 'Partition column values of incorrect type'
	ErrWrongTypeColumnValue = ClassDDL.NewStd(mysql.ErrWrongTypeColumnValue)
	// ErrValuesIsNotIntType returns 'VALUES value for partition '%-.64s' must have type INT'
	ErrValuesIsNotIntType = ClassDDL.NewStd(mysql.ErrValuesIsNotIntType)
	// ErrFunctionalIndexPrimaryKey returns 'The primary key cannot be a functional index'
	ErrFunctionalIndexPrimaryKey = ClassDDL.NewStd(mysql.ErrFunctionalIndexPrimaryKey)
	// ErrFunctionalIndexOnField returns 'Functional index on a column is not supported. Consider using a regular index instead'
	ErrFunctionalIndexOnField = ClassDDL.NewStd(mysql.ErrFunctionalIndexOnField)
	// ErrInvalidAutoRandom returns when auto_random is used incorrectly.
	ErrInvalidAutoRandom = ClassDDL.NewStd(mysql.ErrInvalidAutoRandom)
	// ErrUnsupportedConstraintCheck returns when use ADD CONSTRAINT CHECK
	ErrUnsupportedConstraintCheck = ClassDDL.NewStd(mysql.ErrUnsupportedConstraintCheck)
	// ErrDerivedMustHaveAlias returns when a sub select statement does not have a table alias.
	ErrDerivedMustHaveAlias = ClassDDL.NewStd(mysql.ErrDerivedMustHaveAlias)
	// ErrNullInValuesLessThan returns when a range partition LESS THAN expression includes a NULL
	ErrNullInValuesLessThan = ClassDDL.NewStd(mysql.ErrNullInValuesLessThan)

	// ErrSequenceRunOut returns when the sequence has been run out.
	ErrSequenceRunOut = ClassDDL.NewStd(mysql.ErrSequenceRunOut)
	// ErrSequenceInvalidData returns when sequence values are conflicting.
	ErrSequenceInvalidData = ClassDDL.NewStd(mysql.ErrSequenceInvalidData)
	// ErrSequenceAccessFail returns when sequences are not able to access.
	ErrSequenceAccessFail = ClassDDL.NewStd(mysql.ErrSequenceAccessFail)
	// ErrNotSequence returns when object is not a sequence.
	ErrNotSequence = ClassDDL.NewStd(mysql.ErrNotSequence)
	// ErrUnknownSequence returns when drop / alter unknown sequence.
	ErrUnknownSequence = ClassDDL.NewStd(mysql.ErrUnknownSequence)
	// ErrSequenceUnsupportedTableOption returns when unsupported table option exists in sequence.
	ErrSequenceUnsupportedTableOption = ClassDDL.NewStd(mysql.ErrSequenceUnsupportedTableOption)
	// ErrColumnTypeUnsupportedNextValue is returned when sequence next value is assigned to unsupported column type.
	ErrColumnTypeUnsupportedNextValue = ClassDDL.NewStd(mysql.ErrColumnTypeUnsupportedNextValue)
	// ErrAddColumnWithSequenceAsDefault is returned when the new added column with sequence's nextval as it's default value.
	ErrAddColumnWithSequenceAsDefault = ClassDDL.NewStd(mysql.ErrAddColumnWithSequenceAsDefault)
	// ErrUnsupportedExpressionIndex is returned when create an expression index without allow-expression-index.
	ErrUnsupportedExpressionIndex = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "creating expression index containing unsafe functions without allow-expression-index in config"), nil))
	// ErrPartitionExchangePartTable is returned when exchange table partition with another table is partitioned.
	ErrPartitionExchangePartTable = ClassDDL.NewStd(mysql.ErrPartitionExchangePartTable)
	// ErrPartitionExchangeTempTable is returned when exchange table partition with a temporary table
	ErrPartitionExchangeTempTable = ClassDDL.NewStd(mysql.ErrPartitionExchangeTempTable)
	// ErrTablesDifferentMetadata is returned when exchanges tables is not compatible.
	ErrTablesDifferentMetadata = ClassDDL.NewStd(mysql.ErrTablesDifferentMetadata)
	// ErrRowDoesNotMatchPartition is returned when the row record of exchange table does not match the partition rule.
	ErrRowDoesNotMatchPartition = ClassDDL.NewStd(mysql.ErrRowDoesNotMatchPartition)
	// ErrPartitionExchangeForeignKey is returned when exchanged normal table has foreign keys.
	ErrPartitionExchangeForeignKey = ClassDDL.NewStd(mysql.ErrPartitionExchangeForeignKey)
	// ErrCheckNoSuchTable is returned when exchanged normal table is view or sequence.
	ErrCheckNoSuchTable = ClassDDL.NewStd(mysql.ErrCheckNoSuchTable)
	// ErrUnsupportedPartitionType is returned when exchange table partition type is not supported.
	ErrUnsupportedPartitionType = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "partition type of table %s when exchanging partition"), nil))
	// ErrPartitionExchangeDifferentOption is returned when attribute does not match between partition table and normal table.
	ErrPartitionExchangeDifferentOption = ClassDDL.NewStd(mysql.ErrPartitionExchangeDifferentOption)
	// ErrTableOptionUnionUnsupported is returned when create/alter table with union option.
	ErrTableOptionUnionUnsupported = ClassDDL.NewStd(mysql.ErrTableOptionUnionUnsupported)
	// ErrTableOptionInsertMethodUnsupported is returned when create/alter table with insert method option.
	ErrTableOptionInsertMethodUnsupported = ClassDDL.NewStd(mysql.ErrTableOptionInsertMethodUnsupported)

	// ErrInvalidPlacementPolicyCheck is returned when txn_scope and commit data changing do not meet the placement policy
	ErrInvalidPlacementPolicyCheck = ClassDDL.NewStd(mysql.ErrPlacementPolicyCheck)

	// ErrPlacementPolicyWithDirectOption is returned when create/alter table with both placement policy and placement options existed.
	ErrPlacementPolicyWithDirectOption = ClassDDL.NewStd(mysql.ErrPlacementPolicyWithDirectOption)

	// ErrPlacementPolicyInUse is returned when placement policy is in use in drop/alter.
	ErrPlacementPolicyInUse = ClassDDL.NewStd(mysql.ErrPlacementPolicyInUse)

	// ErrMultipleDefConstInListPart returns multiple definition of same constant in list partitioning.
	ErrMultipleDefConstInListPart = ClassDDL.NewStd(mysql.ErrMultipleDefConstInListPart)

	// ErrTruncatedWrongValue is returned when data has been truncated during conversion.
	ErrTruncatedWrongValue = ClassDDL.NewStd(mysql.ErrTruncatedWrongValue)

	// ErrWarnDataOutOfRange is returned when the value in a numeric column that is outside the permissible range of the column data type.
	// See https://dev.mysql.com/doc/refman/5.5/en/out-of-range-and-overflow.html for details
	ErrWarnDataOutOfRange = ClassDDL.NewStd(mysql.ErrWarnDataOutOfRange)

	// ErrTooLongValueForType is returned when the individual enum element length is too long.
	ErrTooLongValueForType = ClassDDL.NewStd(mysql.ErrTooLongValueForType)

	// ErrUnknownEngine is returned when the table engine is unknown.
	ErrUnknownEngine = ClassDDL.NewStd(mysql.ErrUnknownStorageEngine)

	// ErrExchangePartitionDisabled is returned when exchange partition is disabled.
	ErrExchangePartitionDisabled = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message("Exchange Partition is disabled, please set 'tidb_enable_exchange_partition' if you need to need to enable it", nil))

	// ErrPartitionNoTemporary returns when partition at temporary mode
	ErrPartitionNoTemporary = ClassDDL.NewStd(mysql.ErrPartitionNoTemporary)

	// ErrOptOnTemporaryTable returns when exec unsupported opt at temporary mode
	ErrOptOnTemporaryTable = ClassDDL.NewStd(mysql.ErrOptOnTemporaryTable)
	// ErrOptOnCacheTable returns when exec unsupported opt at cache mode
	ErrOptOnCacheTable = ClassDDL.NewStd(mysql.ErrOptOnCacheTable)
	// ErrUnsupportedOnCommitPreserve returns when exec unsupported opt on commit preserve
	ErrUnsupportedOnCommitPreserve = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message("TiDB doesn't support ON COMMIT PRESERVE ROWS for now", nil))
	// ErrUnsupportedClusteredSecondaryKey returns when exec unsupported clustered secondary key
	ErrUnsupportedClusteredSecondaryKey = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message("CLUSTERED/NONCLUSTERED keyword is only supported for primary key", nil))

	// ErrUnsupportedLocalTempTableDDL returns when ddl operation unsupported for local temporary table
	ErrUnsupportedLocalTempTableDDL = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message("TiDB doesn't support %s for local temporary table", nil))
	// ErrInvalidAttributesSpec is returned when meeting invalid attributes.
	ErrInvalidAttributesSpec = ClassDDL.NewStd(mysql.ErrInvalidAttributesSpec)
	// ErrFunctionalIndexOnJSONOrGeometryFunction returns when creating expression index and the type of the expression is JSON.
	ErrFunctionalIndexOnJSONOrGeometryFunction = ClassDDL.NewStd(mysql.ErrFunctionalIndexOnJSONOrGeometryFunction)
	// ErrDependentByFunctionalIndex returns when the dropped column depends by expression index.
	ErrDependentByFunctionalIndex = ClassDDL.NewStd(mysql.ErrDependentByFunctionalIndex)
	// ErrFunctionalIndexOnBlob when the expression of expression index returns blob or text.
	ErrFunctionalIndexOnBlob = ClassDDL.NewStd(mysql.ErrFunctionalIndexOnBlob)
	// ErrDependentByPartitionFunctional returns when the dropped column depends by expression partition.
	ErrDependentByPartitionFunctional = ClassDDL.NewStd(mysql.ErrDependentByPartitionFunctional)

	// ErrUnsupportedAlterTableSpec means we don't support this alter table specification (i.e. unknown)
	ErrUnsupportedAlterTableSpec = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "Unsupported/unknown ALTER TABLE specification"), nil))
	// ErrGeneralUnsupportedDDL as a generic error to customise by argument
	ErrGeneralUnsupportedDDL = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "%s"), nil))

	// ErrAutoConvert when auto convert happens
	ErrAutoConvert = ClassDDL.NewStd(mysql.ErrAutoConvert)
	// ErrWrongStringLength when UserName or HostName is too long
	ErrWrongStringLength = ClassDDL.NewStd(mysql.ErrWrongStringLength)

	// ErrBinlogUnsafeSystemFunction when use a system function that may return a different value on the slave.
	ErrBinlogUnsafeSystemFunction = ClassDDL.NewStd(mysql.ErrBinlogUnsafeSystemFunction)

	// ErrDDLJobNotFound indicates the job id was not found.
	ErrDDLJobNotFound = ClassDDL.NewStd(mysql.ErrDDLJobNotFound)
	// ErrCancelFinishedDDLJob returns when cancel a finished ddl job.
	ErrCancelFinishedDDLJob = ClassDDL.NewStd(mysql.ErrCancelFinishedDDLJob)
	// ErrCannotCancelDDLJob returns when cancel a almost finished ddl job, because cancel in now may cause data inconsistency.
	ErrCannotCancelDDLJob = ClassDDL.NewStd(mysql.ErrCannotCancelDDLJob)
	// ErrCannotPauseDDLJob returns when the State is not qualified to be paused.
	ErrCannotPauseDDLJob = ClassDDL.NewStd(mysql.ErrCannotPauseDDLJob)
	// ErrCannotResumeDDLJob returns  when the State is not qualified to be resumed.
	ErrCannotResumeDDLJob = ClassDDL.NewStd(mysql.ErrCannotResumeDDLJob)
	// ErrDDLSetting returns when failing to enable/disable DDL.
	ErrDDLSetting = ClassDDL.NewStd(mysql.ErrDDLSetting)
	// ErrIngestFailed returns when the DDL ingest job is failed.
	ErrIngestFailed = ClassDDL.NewStd(mysql.ErrIngestFailed)
	// ErrIngestCheckEnvFailed returns when the DDL ingest env is failed to init.
	ErrIngestCheckEnvFailed = ClassDDL.NewStd(mysql.ErrIngestCheckEnvFailed)

	// ErrColumnInChange indicates there is modification on the column in parallel.
	ErrColumnInChange = ClassDDL.NewStd(mysql.ErrColumnInChange)

	// ErrAlterTiFlashModeForTableWithoutTiFlashReplica returns when set tiflash mode on table whose tiflash_replica is null or tiflash_replica_count = 0
	ErrAlterTiFlashModeForTableWithoutTiFlashReplica = ClassDDL.NewStdErr(0, parser_mysql.Message("TiFlash mode will take effect after at least one TiFlash replica is set for the table", nil))
	// ErrUnsupportedTiFlashOperationForSysOrMemTable means we don't support the alter tiflash related action(e.g. set tiflash mode, set tiflash replica) for system table.
	ErrUnsupportedTiFlashOperationForSysOrMemTable = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "`set TiFlash replica` settings for system table and memory table"), nil))
	// ErrUnsupportedTiFlashOperationForUnsupportedCharsetTable is used when alter alter tiflash related action(e.g. set tiflash mode, set tiflash replica) with unsupported charset.
	ErrUnsupportedTiFlashOperationForUnsupportedCharsetTable = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation, parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw, "`set TiFlash replica` settings for table contains %s charset"), nil))
	// ErrTiFlashBackfillIndex is the error that tiflash backfill the index failed.
	ErrTiFlashBackfillIndex = ClassDDL.NewStdErr(mysql.ErrTiFlashBackfillIndex,
		parser_mysql.Message(mysql.MySQLErrName[mysql.ErrTiFlashBackfillIndex].Raw, nil))

	// ErrDropIndexNeededInForeignKey returns when drop index which is needed in foreign key.
	ErrDropIndexNeededInForeignKey = ClassDDL.NewStd(mysql.ErrDropIndexNeededInForeignKey)
	// ErrForeignKeyCannotDropParent returns when drop table which has foreign key referred.
	ErrForeignKeyCannotDropParent = ClassDDL.NewStd(mysql.ErrForeignKeyCannotDropParent)
	// ErrTruncateIllegalForeignKey returns when truncate table which has foreign key referred.
	ErrTruncateIllegalForeignKey = ClassDDL.NewStd(mysql.ErrTruncateIllegalForeignKey)
	// ErrForeignKeyColumnCannotChange returns when change column which used by foreign key.
	ErrForeignKeyColumnCannotChange = ClassDDL.NewStd(mysql.ErrForeignKeyColumnCannotChange)
	// ErrForeignKeyColumnCannotChangeChild returns when change child table's column which used by foreign key.
	ErrForeignKeyColumnCannotChangeChild = ClassDDL.NewStd(mysql.ErrForeignKeyColumnCannotChangeChild)
	// ErrNoReferencedRow2 returns when there are rows in child table don't have related foreign key value in refer table.
	ErrNoReferencedRow2 = ClassDDL.NewStd(mysql.ErrNoReferencedRow2)

	// ErrUnsupportedColumnInTTLConfig returns when a column type is not expected in TTL config
	ErrUnsupportedColumnInTTLConfig = ClassDDL.NewStd(mysql.ErrUnsupportedColumnInTTLConfig)
	// ErrTTLColumnCannotDrop returns when a column is dropped while referenced by TTL config
	ErrTTLColumnCannotDrop = ClassDDL.NewStd(mysql.ErrTTLColumnCannotDrop)
	// ErrSetTTLOptionForNonTTLTable returns when the `TTL_ENABLE` or `TTL_JOB_INTERVAL` option is set on a non-TTL table
	ErrSetTTLOptionForNonTTLTable = ClassDDL.NewStd(mysql.ErrSetTTLOptionForNonTTLTable)
	// ErrTempTableNotAllowedWithTTL returns when setting TTL config for a temp table
	ErrTempTableNotAllowedWithTTL = ClassDDL.NewStd(mysql.ErrTempTableNotAllowedWithTTL)
	// ErrUnsupportedTTLReferencedByFK returns when the TTL config is set for a table referenced by foreign key
	ErrUnsupportedTTLReferencedByFK = ClassDDL.NewStd(mysql.ErrUnsupportedTTLReferencedByFK)
	// ErrUnsupportedPrimaryKeyTypeWithTTL returns when create or alter a table with TTL options but the primary key is not supported
	ErrUnsupportedPrimaryKeyTypeWithTTL = ClassDDL.NewStd(mysql.ErrUnsupportedPrimaryKeyTypeWithTTL)

	// ErrNotSupportedYet returns when tidb does not support this feature.
	ErrNotSupportedYet = ClassDDL.NewStd(mysql.ErrNotSupportedYet)

	// ErrColumnCheckConstraintReferOther is returned when create column check constraint referring other column.
	ErrColumnCheckConstraintReferOther = ClassDDL.NewStd(mysql.ErrColumnCheckConstraintReferencesOtherColumn)
	// ErrTableCheckConstraintReferUnknown is returned when create table check constraint referring non-existing column.
	ErrTableCheckConstraintReferUnknown = ClassDDL.NewStd(mysql.ErrTableCheckConstraintReferUnknown)
	// ErrConstraintNotFound is returned for dropping a non-existent constraint.
	ErrConstraintNotFound = ClassDDL.NewStd(mysql.ErrConstraintNotFound)
	// ErrCheckConstraintIsViolated is returned for violating an existent check constraint.
	ErrCheckConstraintIsViolated = ClassDDL.NewStd(mysql.ErrCheckConstraintViolated)
	// ErrCheckConstraintNamedFuncIsNotAllowed is returned for not allowed function with name.
	ErrCheckConstraintNamedFuncIsNotAllowed = ClassDDL.NewStd(mysql.ErrCheckConstraintNamedFunctionIsNotAllowed)
	// ErrCheckConstraintFuncIsNotAllowed is returned for not allowed function.
	ErrCheckConstraintFuncIsNotAllowed = ClassDDL.NewStd(mysql.ErrCheckConstraintFunctionIsNotAllowed)
	// ErrCheckConstraintVariables is returned for referring user or system variables.
	ErrCheckConstraintVariables = ClassDDL.NewStd(mysql.ErrCheckConstraintVariables)
	// ErrCheckConstraintRefersAutoIncrementColumn is returned for referring auto-increment columns.
	ErrCheckConstraintRefersAutoIncrementColumn = ClassDDL.NewStd(mysql.ErrCheckConstraintRefersAutoIncrementColumn)
	// ErrCheckConstraintUsingFKReferActionColumn is returned for referring foreign key columns.
	ErrCheckConstraintUsingFKReferActionColumn = ClassDDL.NewStd(mysql.ErrCheckConstraintClauseUsingFKReferActionColumn)
	// ErrNonBooleanExprForCheckConstraint is returned for non bool expression.
	ErrNonBooleanExprForCheckConstraint = ClassDDL.NewStd(mysql.ErrNonBooleanExprForCheckConstraint)
	// ErrWarnDeprecatedIntegerDisplayWidth share the same code 1681, and it will be returned when length is specified in integer.
	ErrWarnDeprecatedIntegerDisplayWidth = ClassDDL.NewStdErr(
		mysql.ErrWarnDeprecatedSyntaxNoReplacement,
		parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrWarnDeprecatedSyntaxNoReplacement].Raw,
			"Integer display width", "",
		), nil),
	)
	// ErrWarnDeprecatedZerofill is for when the deprectated zerofill attribute is used
	ErrWarnDeprecatedZerofill = ClassDDL.NewStdErr(
		mysql.ErrWarnDeprecatedSyntaxNoReplacement,
		parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrWarnDeprecatedSyntaxNoReplacement].Raw,
			"The ZEROFILL attribute",
			" Use the LPAD function to zero-pad numbers, or store the formatted numbers in a CHAR column.",
		), nil),
	)
	// ErrCheckConstraintDupName is for duplicate check constraint names
	ErrCheckConstraintDupName = ClassDDL.NewStd(mysql.ErrCheckConstraintDupName)
	// ErrUnsupportedDistTask is for `tidb_enable_dist_task enabled` but `tidb_ddl_enable_fast_reorg` disabled.
	ErrUnsupportedDistTask = ClassDDL.NewStdErr(mysql.ErrUnsupportedDDLOperation,
		parser_mysql.Message(fmt.Sprintf(mysql.MySQLErrName[mysql.ErrUnsupportedDDLOperation].Raw,
			"tidb_enable_dist_task setting. To utilize distributed task execution, please enable tidb_ddl_enable_fast_reorg first."), nil))
	// ErrGlobalIndexNotExplicitlySet is for Global index when not explicitly said GLOBAL, including UPDATE INDEXES
	ErrGlobalIndexNotExplicitlySet = ClassDDL.NewStd(mysql.ErrGlobalIndexNotExplicitlySet)
	// ErrWarnGlobalIndexNeedManuallyAnalyze is used for global indexes,
	// which cannot trigger automatic analysis when it contains prefix columns or virtual generated columns.
	ErrWarnGlobalIndexNeedManuallyAnalyze = ClassDDL.NewStd(mysql.ErrWarnGlobalIndexNeedManuallyAnalyze)

	// ErrEngineAttributeInvalidFormat is returned when meeting invalid format of engine attribute.
	ErrEngineAttributeInvalidFormat = ClassDDL.NewStd(mysql.ErrEngineAttributeInvalidFormat)
	// ErrStorageClassInvalidSpec is reserved for future use.
	ErrStorageClassInvalidSpec = ClassDDL.NewStd(mysql.ErrStorageClassInvalidSpec)
)

// ReorgRetryableErrCodes are the error codes that are retryable for reorganization.
var ReorgRetryableErrCodes = map[uint16]struct{}{
	mysql.ErrPDServerTimeout:           {},
	mysql.ErrTiKVServerTimeout:         {},
	mysql.ErrTiKVServerBusy:            {},
	mysql.ErrResolveLockTimeout:        {},
	mysql.ErrRegionUnavailable:         {},
	mysql.ErrTxnAbortedByGC:            {},
	mysql.ErrWriteConflict:             {},
	mysql.ErrTiKVStoreLimit:            {},
	mysql.ErrTiKVStaleCommand:          {},
	mysql.ErrTiKVMaxTimestampNotSynced: {},
	mysql.ErrTiFlashServerTimeout:      {},
	mysql.ErrTiFlashServerBusy:         {},
	mysql.ErrInfoSchemaExpired:         {},
	mysql.ErrInfoSchemaChanged:         {},
	mysql.ErrWriteConflictInTiDB:       {},
	mysql.ErrTxnRetryable:              {},
	mysql.ErrNotOwner:                  {},
	mysql.ErrInvalidSplitRegionRanges:  {}, // PD client returns regions with no leader.

	// Temporary network partitioning may cause pk commit failure.
	uint16(terror.CodeResultUndetermined): {},
}

// ReorgRetryableErrMsgs are the error messages that are retryable for reorganization.
var ReorgRetryableErrMsgs = []string{
	"context deadline exceeded",
	"requested lease not found",
}
