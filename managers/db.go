package managers

import (
	"encoding/base64"
	"log/slog"
	"strconv"
	"sync"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB(wg *sync.WaitGroup) {
	defer wg.Done()

	var dialector gorm.Dialector

	if Config.PG.URL != "" {
		dialector = postgres.Open(Config.PG.URL)
	} else {
		dialector = sqlite.Open("binran.db")
	}

	var err error
	if DB, err = gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}); err != nil {
		panic(err)
	}

	if Config.PG.URL == "" {
		DB.AutoMigrate(&MakerSequence{})
	}

	slog.Info("Have connected to database")
}

func IDToString(id uint) string {
	return strconv.Itoa(int(id))
}

func IDToBase64(id uint) string {
	return base64.RawURLEncoding.EncodeToString([]byte{byte(id)})
}

func StringToID(id string) (uint, error) {
	i, err := strconv.ParseInt(id, 10, 32)
	return uint(i), err
}

type MakerSequence struct {
	Name string `gorm:"primary_key"`
	Seq  uint64
}

func IDMakerRegister(table string) {
	DB.Create(&MakerSequence{Name: table})
}

func IDMaker(table string) (uint, error) {
	var (
		id  uint
		err error
	)

	if Config.PG.URL != "" {
		err = DB.Raw("SELECT nextval('" + table + "_id_seq')").Scan(&id).Error
	} else {
		err = DB.Raw("UPDATE maker_sequences SET seq + 1 WHERE name = ? RETURNING seq", table).Scan(&id).Error
	}

	return id, err
}
