package sqlserver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jasonlabz/dbutil/dboperator"
	"github.com/jasonlabz/dbutil/dbx"
)

const DBTypeSqlserver dbx.DBType = dbx.DBTypeSqlserver

func NewSqlserverOperator() dboperator.IOperator {
	return &SqlServerOperator{}
}

type SqlServerOperator struct{}

func (s SqlServerOperator) GetDB(name string) (*dbx.DBWrapper, error) {
	return dbx.GetDB(name)
}

func (s SqlServerOperator) Open(config *dbx.Config) error {
	return dbx.InitConfig(config)
}

func (s SqlServerOperator) Ping(dbName string) error {
	return dbx.Ping(dbName)
}

func (s SqlServerOperator) Close(dbName string) error {
	return dbx.Close(dbName)
}

func (s SqlServerOperator) GetDataBySQL(ctx context.Context, dbName, sqlStatement string) (rows []map[string]interface{}, err error) {
	rows = make([]map[string]interface{}, 0)
	db, err := dbx.GetDB(dbName)
	if err != nil {
		return
	}
	err = db.DB.WithContext(ctx).
		Raw(sqlStatement).
		Find(&rows).Error
	return
}

func (s SqlServerOperator) GetTableData(ctx context.Context, dbName, schemaName, tableName string, pageInfo *dboperator.Pagination) (rows []map[string]interface{}, err error) {
	rows = make([]map[string]interface{}, 0)
	db, err := dbx.GetDB(dbName)
	if err != nil {
		return
	}
	queryTable := fmt.Sprintf("\"%s\"", tableName)
	if schemaName != "" {
		queryTable = fmt.Sprintf("\"%s\".\"%s\"", schemaName, tableName)
	}
	var count int64
	err = db.DB.WithContext(ctx).
		Table(queryTable).
		Count(&count).
		Offset(int(pageInfo.GetOffset())).
		Limit(int(pageInfo.PageSize)).
		Find(&rows).Error
	pageInfo.Total = count
	pageInfo.SetPageCount()
	return
}

func (s SqlServerOperator) GetTablesUnderSchema(ctx context.Context, dbName string, schemas []string) (dbTableMap map[string]*dboperator.LogicDBInfo, err error) {
	dbTableMap = make(map[string]*dboperator.LogicDBInfo)
	if dbName == "" {
		err = errors.New("empty dnName")
		return
	}
	for index, schema := range schemas {
		schemas[index] = "'" + schema + "'"
	}
	gormDBTables := make([]*dboperator.GormDBTable, 0)
	db, err := dbx.GetDB(dbName)
	if err != nil {
		return
	}
	db.DB.WithContext(ctx).
		Raw("select  " +
			"a.name AS table_name, " +
			"b.name as table_schema, " +
			"CONVERT(NVARCHAR(100),isnull(c.[value],'-')) AS comments " +
			"FROM sys.tables a " +
			"LEFT JOIN sys.schemas b " +
			"ON a.schema_id = b.schema_id " +
			"LEFT JOIN sys.extended_properties c " +
			"ON (a.object_id = c.major_id AND c.minor_id = 0) " +
			"WHERE b.name IN (" + strings.Join(schemas, ",") + ") " +
			"ORDER BY b.name,a.name").
		Find(&gormDBTables)
	if len(gormDBTables) == 0 {
		return
	}
	for _, row := range gormDBTables {
		if logicDBInfo, ok := dbTableMap[row.TableSchema]; !ok {
			dbTableMap[row.TableSchema] = &dboperator.LogicDBInfo{
				SchemaName: row.TableSchema,
				TableInfoList: []*dboperator.TableInfo{{
					TableName: row.TableName,
					Comment:   row.Comments,
				}},
			}
		} else {
			logicDBInfo.TableInfoList = append(logicDBInfo.TableInfoList,
				&dboperator.TableInfo{
					TableName: row.TableName,
					Comment:   row.Comments,
				})
		}
	}
	return
}

func (s SqlServerOperator) GetTablesUnderDB(ctx context.Context, dbName string) (dbTableMap map[string]*dboperator.LogicDBInfo, err error) {
	dbTableMap = make(map[string]*dboperator.LogicDBInfo)
	if dbName == "" {
		err = errors.New("empty dnName")
		return
	}
	gormDBTables := make([]*dboperator.GormDBTable, 0)
	db, err := dbx.GetDB(dbName)
	if err != nil {
		return
	}
	db.DB.WithContext(ctx).
		Raw("select  " +
			"a.name AS table_name, " +
			"b.name as table_schema, " +
			"CONVERT(NVARCHAR(100),isnull(c.[value],'-')) AS comments " +
			"FROM sys.tables a " +
			"LEFT JOIN sys.schemas b " +
			"ON a.schema_id = b.schema_id " +
			"LEFT JOIN sys.extended_properties c " +
			"ON (a.object_id = c.major_id AND c.minor_id = 0) " +
			"WHERE b.name not like 'db_%' and  b.name NOT IN ('sys','INFORMATION_SCHEMA') " +
			"ORDER BY b.name,a.name").
		Find(&gormDBTables)
	if len(gormDBTables) == 0 {
		return
	}
	for _, row := range gormDBTables {
		if logicDBInfo, ok := dbTableMap[row.TableSchema]; !ok {
			dbTableMap[row.TableSchema] = &dboperator.LogicDBInfo{
				SchemaName: row.TableSchema,
				TableInfoList: []*dboperator.TableInfo{{
					TableName: row.TableName,
					Comment:   row.Comments,
				}},
			}
		} else {
			logicDBInfo.TableInfoList = append(logicDBInfo.TableInfoList,
				&dboperator.TableInfo{
					TableName: row.TableName,
					Comment:   row.Comments,
				})
		}
	}
	return
}

func (s SqlServerOperator) GetColumns(ctx context.Context, dbName string) (dbTableColMap map[string]map[string]*dboperator.TableColInfo, err error) {
	dbTableColMap = make(map[string]map[string]*dboperator.TableColInfo, 0)
	if dbName == "" {
		err = errors.New("empty dnName")
		return
	}
	gormTableColumns := make([]*dboperator.GormTableColumn, 0)
	db, err := dbx.GetDB(dbName)
	if err != nil {
		return
	}
	db.DB.WithContext(ctx).
		Raw("SELECT TABLE_SCHEMA as table_schema, " +
			"TABLE_NAME as table_name, " +
			"COLUMN_NAME as column_name, " +
			"DATA_TYPE as data_type " +
			"FROM INFORMATION_SCHEMA.Columns " +
			"WHERE TABLE_SCHEMA NOT IN ('sys','INFORMATION_SCHEMA') " +
			"ORDER BY TABLE_NAME, COLUMN_NAME").
		Find(&gormTableColumns)
	if len(gormTableColumns) == 0 {
		return
	}

	for _, row := range gormTableColumns {
		if dbTableColInfoMap, ok := dbTableColMap[row.TableSchema]; !ok {
			dbTableColMap[row.TableSchema] = map[string]*dboperator.TableColInfo{
				row.TableName: {
					TableName: row.TableName,
					ColumnInfoList: []*dboperator.ColumnInfo{{
						ColumnName: row.ColumnName,
						Comment:    row.Comments,
						DataType:   row.DataType,
					}},
				},
			}
		} else if tableColInfo, ok_ := dbTableColInfoMap[row.TableName]; !ok_ {
			dbTableColInfoMap[row.TableName] = &dboperator.TableColInfo{
				TableName: row.TableName,
				ColumnInfoList: []*dboperator.ColumnInfo{{
					ColumnName: row.ColumnName,
					Comment:    row.Comments,
					DataType:   row.DataType,
				}},
			}
		} else {
			tableColInfo.ColumnInfoList = append(tableColInfo.ColumnInfoList, &dboperator.ColumnInfo{
				ColumnName: row.ColumnName,
				Comment:    row.Comments,
				DataType:   row.DataType,
			})
		}
	}
	return
}

func (s SqlServerOperator) GetColumnsUnderTables(ctx context.Context, dbName, logicDBName string, tableNames []string) (tableColMap map[string]*dboperator.TableColInfo, err error) {
	tableColMap = make(map[string]*dboperator.TableColInfo, 0)
	if dbName == "" {
		err = errors.New("empty dnName")
		return
	}
	if len(tableNames) == 0 {
		err = errors.New("empty tableNames")
		return
	}

	gormTableColumns := make([]*dboperator.GormTableColumn, 0)
	db, err := dbx.GetDB(dbName)
	if err != nil {
		return
	}
	db.DB.WithContext(ctx).
		Raw("SELECT TABLE_SCHEMA as table_schema, "+
			"TABLE_NAME as table_name, "+
			"COLUMN_NAME as column_name, "+
			"DATA_TYPE as data_type "+
			"FROM INFORMATION_SCHEMA.Columns "+
			"WHERE TABLE_SCHEMA = ? "+
			"AND TABLE_NAME IN ? "+
			"ORDER BY TABLE_NAME, COLUMN_NAME", logicDBName, tableNames).
		Find(&gormTableColumns)
	if len(gormTableColumns) == 0 {
		return
	}

	for _, row := range gormTableColumns {
		if tableColInfo, ok := tableColMap[row.TableName]; !ok {
			tableColMap[row.TableName] = &dboperator.TableColInfo{
				TableName: row.TableName,
				ColumnInfoList: []*dboperator.ColumnInfo{{
					ColumnName: row.ColumnName,
					Comment:    row.Comments,
					DataType:   row.DataType,
				}},
			}
		} else {
			tableColInfo.ColumnInfoList = append(tableColInfo.ColumnInfoList, &dboperator.ColumnInfo{
				ColumnName: row.ColumnName,
				Comment:    row.Comments,
				DataType:   row.DataType,
			})
		}
	}
	return
}

func (s SqlServerOperator) CreateSchema(ctx context.Context, dbName, schemaName, commentInfo string) (err error) {
	if dbName == "" {
		err = errors.New("empty dnName")
		return
	}
	if commentInfo == "" {
		commentInfo = schemaName
	}
	db, err := dbx.GetDB(dbName)
	if err != nil {
		return
	}
	err = db.DB.WithContext(ctx).Exec("create schema " + schemaName).Error
	if err != nil {
		return
	}
	return
}

func (s SqlServerOperator) ExecuteDDL(ctx context.Context, dbName, ddlStatement string) (err error) {
	if dbName == "" {
		err = errors.New("empty dnName")
		return
	}
	db, err := dbx.GetDB(dbName)
	if err != nil {
		return
	}
	err = db.DB.WithContext(ctx).Exec(ddlStatement).Error
	if err != nil {
		return
	}
	return
}

func init() {
	err := dboperator.RegisterDS(DBTypeSqlserver, NewSqlserverOperator())
	if err != nil {
		panic(err)
	}
}