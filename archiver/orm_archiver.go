package archiver

import (
	"encoding/json"
	"log"
	"math/big"
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
	case "sqlite":
		db, err := gorm.Open(sqlite.Open(conf.DSN), &gorm.Config{})
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

func NewORMArchiverFromConfig(conf *ORMArchiverConfig) (*ORMArchiver, error) {
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
	case "sqlite":
		db, err := gorm.Open(sqlite.Open(conf.DSN), &gorm.Config{})
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

func (oa *ORMArchiver) GetByCommitment(args *types.GetBlockCommitmentArgs, reply *types.SequencerWarpBlockResponse, blockParser chain.Parser) error {

	if args.First < 1 {
		return nil
	}
	type BlockInfoWithParent struct {
		BlockId   string `json:"id"`
		Parent    string `json:"parent"`
		Timestamp int64  `json:"timestamp"`
		L1Head    uint64 `json:"l1_head"`
		Height    uint64 `json:"height"`
	}

	//TODO check if this works

	var blocks []BlockInfoWithParent

	if err := oa.db.Raw("SELECT BlockId, Timestamp, L1Head, Height FROM DBBlock WHERE Height >= ? AND Height < ? ORDER BY Height LIMIT ?", args.First, args.CurrentHeight, args.MaxBlocks).Scan(&blocks).Error; err != nil {
		return err
	}

	blocksCommitment := make([]types.SequencerWarpBlock, 0)

	for _, blk := range blocks {

		id, err := ids.FromString(blk.BlockId)
		if err != nil {
			return err
		}

		header := &types.Header{
			Height:    blk.Height,
			Timestamp: uint64(blk.Timestamp),
			L1Head:    uint64(blk.L1Head),
			TransactionsRoot: types.NmtRoot{
				Root: id[:],
			},
		}

		comm := header.Commit()

		idParent, err := ids.FromString(blk.Parent)
		if err != nil {
			return err
		}

		parentRoot := types.NewU256().SetBytes(idParent)
		bigParentRoot := parentRoot.Int

		blocksCommitment = append(blocksCommitment, types.SequencerWarpBlock{
			BlockId:    id.String(),
			Timestamp:  blk.Timestamp,
			L1Head:     uint64(blk.L1Head),
			Height:     big.NewInt(int64(blk.Height)),
			BlockRoot:  &comm.Uint256().Int,
			ParentRoot: &bigParentRoot,
		})

	}

	reply.Blocks = blocksCommitment
	return nil
}
