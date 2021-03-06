package gorm

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	IGNORE = "IGNORE"
)

// Define callbacks for creating
func init() {
	DefaultCallback.Create().Register("gorm:begin_transaction", beginTransactionCallback)
	DefaultCallback.Create().Register("gorm:before_create", beforeCreateCallback)
	DefaultCallback.Create().Register("gorm:save_before_associations", saveBeforeAssociationsCallback)
	DefaultCallback.Create().Register("gorm:update_time_stamp", updateTimeStampForCreateCallback)
	DefaultCallback.Create().Register("gorm:create", createCallback)
	DefaultCallback.Create().Register("gorm:force_reload_after_create", forceReloadAfterCreateCallback)
	DefaultCallback.Create().Register("gorm:save_after_associations", saveAfterAssociationsCallback)
	DefaultCallback.Create().Register("gorm:after_create", afterCreateCallback)
	DefaultCallback.Create().Register("gorm:commit_or_rollback_transaction", commitOrRollbackTransactionCallback)
}

// beforeCreateCallback will invoke `BeforeSave`, `BeforeCreate` method before creating
func beforeCreateCallback(scope *Scope) {
	if !scope.HasError() {
		scope.CallMethod("BeforeSave")
	}
	if !scope.HasError() {
		scope.CallMethod("BeforeCreate")
	}
}

// updateTimeStampForCreateCallback will set `CreatedAt`, `UpdatedAt` when creating
func updateTimeStampForCreateCallback(scope *Scope) {
	if !scope.HasError() {
		now := scope.db.nowFunc()

		if createdAtField, ok := scope.FieldByName("CreatedAt"); ok {
			if createdAtField.IsBlank {
				createdAtField.Set(now)
			}
		}

		if updatedAtField, ok := scope.FieldByName("UpdatedAt"); ok {
			if updatedAtField.IsBlank {
				updatedAtField.Set(now)
			}
		}
	}
}

// createCallback the callback used to insert data into database
func createCallback(scope *Scope) {
	if scope.HasError() {
		return
	}
	defer scope.trace(NowFunc())

	var (
		columns, placeholders        []string
		blankColumnsWithDefaultValue []string
	)

	// Set columns; Add placeholders and vars for `value_list`
	var (
		columnsString      string
		placeholdersString string
	)
	if values, ok := scope.Get("gorm:create_many"); ok {
		// CreateMany
		for _, field := range scope.Fields() {
			if !field.IsPrimaryKey || !field.IsBlank {
				columns = append(columns, field.DBName)
			}
		}
		createMany := values.([](map[string]interface{}))
		var placeholdersStrings []string
		firstObjLength := len(createMany[0])
		for _, obj := range createMany {
			if len(obj) != firstObjLength {
				scope.Err(errors.New("createMany objects should have the same fields"))
				return
			}
			placeholders = []string{}
			for _, column := range columns {
				if fieldValue, ok := obj[column]; ok {
					placeholders = append(placeholders, scope.AddToVars(fieldValue))
				} else {
					field, _ := scope.FieldByName(column)
					placeholders = append(placeholders, scope.AddToVars(field.Field.Interface()))
				}
			}
			placeholdersStrings = append(placeholdersStrings, "("+strings.Join(placeholders, ",")+")")
		}
		for index, column := range columns {
			columns[index] = scope.Quote(column)
		}
		columnsString = strings.Join(columns, ",")
		placeholdersString = strings.Join(placeholdersStrings, ",")
	} else {
		// Normal
		for _, field := range scope.Fields() {
			if scope.changeableField(field) {
				if field.IsNormal && !field.IsIgnored {
					if field.IsBlank && field.HasDefaultValue {
						blankColumnsWithDefaultValue = append(blankColumnsWithDefaultValue, scope.Quote(field.DBName))
						scope.InstanceSet("gorm:blank_columns_with_default_value", blankColumnsWithDefaultValue)
					} else if !field.IsPrimaryKey || !field.IsBlank {
						columns = append(columns, scope.Quote(field.DBName))
						placeholders = append(placeholders, scope.AddToVars(field.Field.Interface()))
					}
				} else if field.Relationship != nil && field.Relationship.Kind == "belongs_to" {
					for _, foreignKey := range field.Relationship.ForeignDBNames {
						if foreignField, ok := scope.FieldByName(foreignKey); ok && !scope.changeableField(foreignField) {
							columns = append(columns, scope.Quote(foreignField.DBName))
							placeholders = append(placeholders, scope.AddToVars(foreignField.Field.Interface()))
						}
					}
				}
			}
		}
		columnsString = strings.Join(columns, ",")
		placeholdersString = "(" + strings.Join(placeholders, ",") + ")"
	}

	var (
		returningColumn = "*"
		quotedTableName = scope.QuotedTableName()
		primaryField    = scope.PrimaryField()
		extraOption     string
		insertModifier  string
	)

	// Add placeholders and vars for `ON CONFLICT`
	if obj, ok := scope.Get("gorm:on_conflict_update"); ok {
		insertStr, ok := scope.Get("gorm:insert_option")
		if !ok {
			scope.Err(errors.New("gorm:insert_option not found"))
			return
		}
		updateMap := obj.(map[string]interface{})
		updateColumns := []string{}
		for field, _ := range updateMap {
			updateColumns = append(updateColumns, field)
		}
		sort.Strings(updateColumns)
		updateSqls := []string{}
		for _, column := range updateColumns {
			value := updateMap[column]
			updateSqls = append(updateSqls, fmt.Sprintf("%v = %v", scope.Quote(column), scope.AddToVars(value)))
		}
		updateSql := strings.Join(updateSqls, ",")
		extraOption = fmt.Sprintf(fmt.Sprint(insertStr), updateSql)
	} else if str, ok := scope.Get("gorm:insert_option"); ok {
		extraOption = fmt.Sprint(str)
	}
	// Set insert_modifier
	if str, ok := scope.Get("gorm:insert_modifier"); ok {
		insertModifier = strings.ToUpper(fmt.Sprint(str))
		if insertModifier == "INTO" {
			insertModifier = ""
		}
	}

	if primaryField != nil {
		returningColumn = scope.Quote(primaryField.DBName)
	}

	// Set `RETURNING`
	lastInsertIDOutputInterstitial := scope.Dialect().LastInsertIDOutputInterstitial(quotedTableName, returningColumn, columns)
	var lastInsertIDReturningSuffix string
	if lastInsertIDOutputInterstitial == "" {
		lastInsertIDReturningSuffix = scope.Dialect().LastInsertIDReturningSuffix(quotedTableName, returningColumn)
	}

	// Set scope.SQL
	if len(columns) == 0 {
		scope.Raw(fmt.Sprintf(
			"INSERT%v INTO %v %v%v%v",
			addExtraSpaceIfExist(insertModifier),
			quotedTableName,
			scope.Dialect().DefaultValueStr(),
			addExtraSpaceIfExist(extraOption),
			addExtraSpaceIfExist(lastInsertIDReturningSuffix),
		))
	} else {
		scope.Raw(fmt.Sprintf(
			"INSERT%v INTO %v (%v)%v VALUES %v%v%v",
			addExtraSpaceIfExist(insertModifier),
			scope.QuotedTableName(),
			columnsString,
			addExtraSpaceIfExist(lastInsertIDOutputInterstitial),
			placeholdersString,
			addExtraSpaceIfExist(extraOption),
			addExtraSpaceIfExist(lastInsertIDReturningSuffix),
		))
	}

	// execute create sql: no primaryField
	if primaryField == nil {
		if result, err := scope.SQLDB().Exec(scope.SQL, scope.SQLVars...); scope.Err(err) == nil {
			// set rows affected count
			scope.db.RowsAffected, _ = result.RowsAffected()

			// set primary value to primary field
			if primaryField != nil && primaryField.IsBlank {
				if primaryValue, err := result.LastInsertId(); scope.Err(err) == nil {
					scope.Err(primaryField.Set(primaryValue))
				}
			}
		}
		return
	}

	// execute create sql: lastInsertID implemention for majority of dialects
	if lastInsertIDReturningSuffix == "" && lastInsertIDOutputInterstitial == "" {
		if result, err := scope.SQLDB().Exec(scope.SQL, scope.SQLVars...); scope.Err(err) == nil {
			// set rows affected count
			scope.db.RowsAffected, _ = result.RowsAffected()

			// set primary value to primary field
			if primaryField != nil && primaryField.IsBlank {
				if primaryValue, err := result.LastInsertId(); scope.Err(err) == nil {
					scope.Err(primaryField.Set(primaryValue))
				}
			}
		}
		return
	}

	// execute create sql: dialects with additional lastInsertID requirements (currently postgres & mssql)
	if primaryField.Field.CanAddr() {
		if err := scope.SQLDB().QueryRow(scope.SQL, scope.SQLVars...).Scan(primaryField.Field.Addr().Interface()); scope.Err(err) == nil {
			primaryField.IsBlank = false
			scope.db.RowsAffected = 1
			if values, ok := scope.Get("gorm:create_many"); ok {
				scope.db.RowsAffected = int64(len(values.([](map[string]interface{}))))
			}
		}
	} else {
		scope.Err(ErrUnaddressable)
	}
	return
}

// forceReloadAfterCreateCallback will reload columns that having default value, and set it back to current object
func forceReloadAfterCreateCallback(scope *Scope) {
	if blankColumnsWithDefaultValue, ok := scope.InstanceGet("gorm:blank_columns_with_default_value"); ok {
		db := scope.DB().New().Table(scope.TableName()).Select(blankColumnsWithDefaultValue.([]string))
		for _, field := range scope.Fields() {
			if field.IsPrimaryKey && !field.IsBlank {
				db = db.Where(fmt.Sprintf("%v = ?", field.DBName), field.Field.Interface())
			}
		}
		db.Scan(scope.Value)
	}
}

// afterCreateCallback will invoke `AfterCreate`, `AfterSave` method after creating
func afterCreateCallback(scope *Scope) {
	if _, ok := scope.Get("gorm:create_many"); ok {
		return
	}
	if !scope.HasError() {
		scope.CallMethod("AfterCreate")
	}
	if !scope.HasError() {
		scope.CallMethod("AfterSave")
	}
}
