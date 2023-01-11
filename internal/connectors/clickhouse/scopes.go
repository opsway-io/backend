package clickhouse

import "gorm.io/gorm"

func Paginated(offset *int, limit *int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if offset != nil {
			db = db.Offset(*offset)
		}

		if limit != nil {
			db = db.Limit(*limit)
		}

		return db
	}
}
