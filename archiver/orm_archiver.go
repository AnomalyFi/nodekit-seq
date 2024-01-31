package archiver

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/nodekit-seq/types"
	"github.com/ava-labs/avalanchego/ids"
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
	BlockId   string `gorm:"index"`
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

func (oa *ORMArchiver) InsertBlock(block *chain.StatelessBlock) error {
	blkID := block.ID()
	blkBytes, err := block.Marshal()
	if err != nil {
		return err
	}

	//TODO need to add L1Head and real Id
	newBlock := DBBlock{
		BlockId:   blkID.String(),
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

func (oa *ORMArchiver) GetBlock(dbBlock *DBBlock, blockParser chain.Parser) (*chain.StatefulBlock, *ids.ID, error) {
	tx := oa.db.Last(dbBlock)
	if tx.Error != nil {
		return nil, nil, tx.Error
	}

	blk, err := chain.UnmarshalBlock(dbBlock.Bytes, blockParser)
	if err != nil {
		return nil, nil, err
	}

	id, err := ids.FromString(dbBlock.BlockId)
	if err != nil {
		return nil, nil, err
	}

	return blk, &id, nil
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

func (oa *ORMArchiver) GetByHeight(height uint64, end int64, blockParser chain.Parser, reply *types.BlockHeadersResponse) error {
	Prev := types.BlockInfo{}

	if height > 1 {
		dbBlock := DBBlock{
			BlockId: "",
			Height:  height - 1,
		}
		tx := oa.db.Last(dbBlock)
		if tx.Error != nil {
			return tx.Error
		}

		blk, err := chain.UnmarshalBlock(dbBlock.Bytes, blockParser)
		if err != nil {
			return err
		}

		Prev = types.BlockInfo{
			BlockId:   dbBlock.BlockId,
			Timestamp: blk.Tmstmp,
			L1Head:    uint64(blk.L1Head),
			Height:    blk.Hght,
		}

	}

	var blocks []types.BlockInfo

	if err := oa.db.Raw("SELECT BlockId, Timestamp, L1Head, Height FROM DBBlock WHERE Height >= ? AND Timestamp < ? ORDER BY Height", height, end).Scan(&blocks).Error; err != nil {
		return err
	}

	var Next types.BlockInfo

	if err := oa.db.Raw("SELECT BlockId, Timestamp, L1Head, Height FROM DBBlock WHERE Timestamp >= ? ORDER BY Height LIMIT 1", end).Scan(&Next).Error; err != nil {
		return err
	}

	//TODO return prev, next, blocks

	reply.From = height
	reply.Blocks = blocks
	reply.Prev = Prev
	reply.Next = Next

	return nil
}

func (oa *ORMArchiver) GetByID(args *types.GetBlockHeadersIDArgs, reply *types.BlockHeadersResponse, blockParser chain.Parser) error {

	var firstBlock uint64

	if args.ID != "" {
		dbBlock := DBBlock{
			BlockId: args.ID,
		}
		tx := oa.db.Last(dbBlock)
		if tx.Error != nil {
			return tx.Error
		}

		blk, err := chain.UnmarshalBlock(dbBlock.Bytes, blockParser)
		if err != nil {
			return err
		}

		firstBlock = blk.Hght
		// Handle hash parameter
		// ...
	} else {
		firstBlock = 1
	}

	Prev := types.BlockInfo{}
	if firstBlock > 1 {
		dbBlock := DBBlock{
			BlockId: "",
			Height:  firstBlock - 1,
		}
		tx := oa.db.Last(dbBlock)
		if tx.Error != nil {
			return tx.Error
		}

		blk, err := chain.UnmarshalBlock(dbBlock.Bytes, blockParser)
		if err != nil {
			return err
		}

		Prev = types.BlockInfo{
			BlockId:   dbBlock.BlockId,
			Timestamp: blk.Tmstmp,
			L1Head:    uint64(blk.L1Head),
			Height:    blk.Hght,
		}
	}

	var blocks []types.BlockInfo

	if err := oa.db.Raw("SELECT BlockId, Timestamp, L1Head, Height FROM DBBlock WHERE Height >= ? AND Timestamp < ? ORDER BY Height", firstBlock, args.End).Scan(&blocks).Error; err != nil {
		return err
	}

	var Next types.BlockInfo

	if err := oa.db.Raw("SELECT BlockId, Timestamp, L1Head, Height FROM DBBlock WHERE Timestamp >= ? ORDER BY Height LIMIT 1", args.End).Scan(&Next).Error; err != nil {
		return err
	}

	reply.From = firstBlock
	reply.Blocks = blocks
	reply.Prev = Prev
	reply.Next = Next

	return nil
}

func (oa *ORMArchiver) GetByStart(args *types.GetBlockHeadersByStartArgs, reply *types.BlockHeadersResponse, blockParser chain.Parser) error {

	var firstBlock uint64

	Prev := types.BlockInfo{}

	//TODO check if this works
	dbBlock := DBBlock{
		Timestamp: args.Start,
	}
	tx := oa.db.Last(dbBlock)
	if tx.Error != nil {
		return tx.Error
	}

	blk, err := chain.UnmarshalBlock(dbBlock.Bytes, blockParser)
	if err != nil {
		return err
	}

	firstBlock = blk.Hght

	if firstBlock > 1 {
		dbBlock := DBBlock{
			BlockId: "",
			Height:  firstBlock - 1,
		}
		tx := oa.db.Last(dbBlock)
		if tx.Error != nil {
			return tx.Error
		}

		blk, err := chain.UnmarshalBlock(dbBlock.Bytes, blockParser)
		if err != nil {
			return err
		}

		Prev = types.BlockInfo{
			BlockId:   dbBlock.BlockId,
			Timestamp: blk.Tmstmp,
			L1Head:    uint64(blk.L1Head),
			Height:    blk.Hght,
		}
	}

	var blocks []types.BlockInfo

	if err := oa.db.Raw("SELECT BlockId, Timestamp, L1Head, Height FROM DBBlock WHERE Height >= ? AND Timestamp < ? ORDER BY Height", firstBlock, args.End).Scan(&blocks).Error; err != nil {
		return err
	}

	var Next types.BlockInfo

	if err := oa.db.Raw("SELECT BlockId, Timestamp, L1Head, Height FROM DBBlock WHERE Timestamp >= ? ORDER BY Height LIMIT 1", args.End).Scan(&Next).Error; err != nil {
		return err
	}

	reply.From = firstBlock
	reply.Blocks = blocks
	reply.Prev = Prev
	reply.Next = Next

	return nil
}
