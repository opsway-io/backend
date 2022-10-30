package postgres

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

func Search(column []string, search *string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if search != nil {
			for i, c := range column {
				if i == 0 {
					db = db.Where(c+" LIKE ?", "%"+*search+"%")
				} else {
					db = db.Or(c+" LIKE ?", "%"+*search+"%")
				}
			}
		}

		return db
	}
}
