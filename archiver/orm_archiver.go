package archiver

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/AnomalyFi/hypersdk/chain"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type ORMArchiverConfig struct {
	Enabled      bool   `json:"enabled"`
	ArchiverType string `json:"archiverType"`
	DSN          string `json:"dsn"`
}

type ORMArchiver struct {
	db *gorm.DB
}

type DBBlock struct {
	gorm.Model
	BlockID   string `gorm:"index"`
	Parent    string
	Timestamp int64
	Height    uint64 `gorm:"index"`

	Bytes []byte
}

func NewORMArchiver(db *gorm.DB) *ORMArchiver {
	db.AutoMigrate(&DBBlock{})

	return &ORMArchiver{
		db: db,
	}
}

func NewORMArchiverFromConfigBytes(configBytes []byte) (*ORMArchiver, error) {
	var conf ORMArchiverConfig
	err := json.Unmarshal(configBytes, &conf)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(conf.ArchiverType) {
	case "postgresql":
		db, err := gorm.Open(postgres.New(postgres.Config{
			DSN:                  conf.DSN,
			PreferSimpleProtocol: true,
		}))
		log.Println("using postgresql as archiver")
		if err != nil {
			return nil, err
		}
		return NewORMArchiver(db), nil
	default:
		db, err := gorm.Open(sqlite.Open("default.db"), &gorm.Config{})
		if err != nil {
			return nil, err
		}
		return NewORMArchiver(db), nil
	}
}

func (oa *ORMArchiver) InsertBlock(block *chain.StatefulBlock) error {
	blkID, err := block.ID()
	if err != nil {
		return err
	}
	blkBytes, err := block.Marshal()
	if err != nil {
		return err
	}

	newBlock := DBBlock{
		BlockID:   blkID.String(),
		Parent:    block.Prnt.String(),
		Timestamp: block.Tmstmp,
		Height:    block.Hght,
		Bytes:     blkBytes,
	}

	tx := oa.db.Begin()
	if err := tx.Create(&newBlock).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

func (oa *ORMArchiver) GetBlock(dbBlock *DBBlock, blockParser chain.Parser) (*chain.StatefulBlock, error) {
	tx := oa.db.Last(dbBlock)
	if tx.Error != nil {
		return nil, tx.Error
	}

	blk, err := chain.UnmarshalBlock(dbBlock.Bytes, blockParser)
	if err != nil {
		return nil, err
	}

	return blk, nil
}

func (oa *ORMArchiver) DeleteBlock(dbBlock *DBBlock) (bool, error) {
	tx := oa.db.Begin()
	if err := tx.Delete(&dbBlock).Error; err != nil {
		return false, err
	}
	if err := tx.Commit().Error; err != nil {
		return false, err
	}

	return true, nil
}
