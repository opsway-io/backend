package seeds

import "gorm.io/gorm"

type Seeder func(db *gorm.DB)
