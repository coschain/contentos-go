package plugins

import (
	"fmt"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/jinzhu/gorm"
	"regexp"
)

func RemoveSQLTables(dbConfig *service_configs.DatabaseConfig) error {
	connStr := fmt.Sprintf("%s:%s@/%s?charset=utf8&parseTime=True&loc=Local", dbConfig.User, dbConfig.Password, dbConfig.Db)
	db, err := gorm.Open(dbConfig.Driver, connStr)
	if err != nil {
		return err
	}
	defer func(){ _ = db.Close() }()

	rows, err := db.Raw("SHOW TABLES").Rows()
	if err != nil {
		return err
	}

	var names []string
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			matched := false
			for _, e := range sTableNamePatterns {
				if matched = e.MatchString(name); matched {
					break
				}
			}
			if matched {
				names = append(names, name)
			}
		}
	}
	_ = rows.Close()

	for _, name := range names {
		if err := db.DropTable(name).Error; err != nil {
			return err
		}
	}
	return nil
}

var sTableNamePatterns = make(map[string]*regexp.Regexp)

func RegisterSQLTableNamePattern(pattern string) {
	p := fmt.Sprintf("^%s$", pattern)
	sTableNamePatterns[p] = regexp.MustCompile(p)
}
