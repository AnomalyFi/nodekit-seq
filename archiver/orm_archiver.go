package archiver

import (
	"encoding/json"
	"fmt"
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
	BlockId   string `gorm:"index;unique"`
	Parent    string
	Timestamp int64
	Height    uint64 `gorm:"index;unique"`
	L1Head    int64

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

	log.Printf("using %s as archiver\n", conf.ArchiverType)
	switch strings.ToLower(conf.ArchiverType) {
	case "postgresql":
		db, err := gorm.Open(postgres.New(postgres.Config{
			DSN:                  conf.DSN,
			PreferSimpleProtocol: true,
		}))
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
	fmt.Printf("inserting block(%d): %s\n", block.Hght, blkID.String())

	//TODO need to add L1Head and real Id
	newBlock := DBBlock{
		BlockId:   blkID.String(),
		Parent:    block.Prnt.String(),
		Timestamp: block.Tmstmp,
		Height:    block.Hght,
		L1Head:    block.L1Head,
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
	tx := oa.db.First(dbBlock)
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

func (oa *ORMArchiver) GetBlockByID(id string, parser chain.Parser) (*chain.StatefulBlock, error) {
	var dbBlock DBBlock
	tx := oa.db.Where("block_id = ?", id).Find(&dbBlock)
	if tx.Error != nil {
		return nil, tx.Error
	}
	fmt.Printf("block height id: %s, wanted: %s\n", dbBlock.BlockId, id)

	blk, err := chain.UnmarshalBlock(dbBlock.Bytes, parser)
	if err != nil {
		return nil, err
	}
	return blk, nil
}

func (oa *ORMArchiver) GetBlockByHeight(height uint64, parser chain.Parser) (*chain.StatefulBlock, error) {
	var dbBlock DBBlock
	tx := oa.db.Where("height = ?", height).Find(&dbBlock)
	if tx.Error != nil {
		return nil, tx.Error
	}

	blk, err := chain.UnmarshalBlock(dbBlock.Bytes, parser)
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

func (oa *ORMArchiver) GetBlockHeadersByHeight(args *types.GetBlockHeadersByHeightArgs) (*types.BlockHeadersResponse, error) {
	ret := new(types.BlockHeadersResponse)

	if args.Height > 1 {
		var dbBlock DBBlock
		tx := oa.db.Where("height = ?", args.Height-1).Find(&dbBlock)
		if tx.Error != nil {
			return nil, tx.Error
		}

		ret.Prev = types.BlockInfo{
			BlockId:   dbBlock.BlockId,
			Timestamp: dbBlock.Timestamp,
			L1Head:    uint64(dbBlock.L1Head),
			Height:    dbBlock.Height,
		}
	}

	var blocks []DBBlock
	res := oa.db.Where("height >= ? AND timestamp < ?", args.Height, args.End).Order("height").Find(&blocks)
	if res.Error != nil {
		return nil, res.Error
	}

	ret.Blocks = make([]types.BlockInfo, 0, len(blocks))
	for _, block := range blocks {
		blkInfo := types.BlockInfo{
			BlockId:   block.BlockId,
			Timestamp: block.Timestamp,
			L1Head:    uint64(block.L1Head),
			Height:    block.Height,
		}
		if len(ret.Blocks) > 0 && blkInfo.Height-1 != ret.Blocks[len(ret.Blocks)-1].Height {
			return nil, fmt.Errorf("queried blocks aren't consecutive, prev: %d, current: %d", blkInfo.Height, ret.Blocks[len(ret.Blocks)-1].Height)
		}
		ret.Blocks = append(ret.Blocks, blkInfo)
	}
	var next DBBlock
	res = oa.db.Where("timestamp >= ?", args.End).Order("height").First(&next)
	if res.Error == nil {
		ret.Next = types.BlockInfo{
			BlockId:   next.BlockId,
			Timestamp: next.Timestamp,
			L1Head:    uint64(next.L1Head),
			Height:    next.Height,
		}
	} else {
		ret.Next = types.BlockInfo{}
	}

	return ret, nil
}

func (oa *ORMArchiver) GetBlockHeadersByID(args *types.GetBlockHeadersIDArgs) (*types.BlockHeadersResponse, error) {
	ret := new(types.BlockHeadersResponse)
	if args.ID == "" {
		return nil, fmt.Errorf("ID in args is not specified")
	}

	var firstBlock DBBlock
	tx := oa.db.Where("block_id", args.ID).Find(&firstBlock)
	if tx.Error != nil {
		return nil, tx.Error
	}

	firstBlockHeight := firstBlock.Height

	ret.Prev = types.BlockInfo{}
	if firstBlockHeight > 1 {
		var prevBlock DBBlock
		tx := oa.db.Where("height = ?", firstBlockHeight-1).Find(&prevBlock)
		if tx.Error != nil {
			return nil, tx.Error
		}

		ret.Prev = types.BlockInfo{
			BlockId:   prevBlock.BlockId,
			Timestamp: prevBlock.Timestamp,
			L1Head:    uint64(prevBlock.L1Head),
			Height:    prevBlock.Height,
		}
	}

	var blocks []DBBlock
	res := oa.db.Where("height >= ? AND timestamp < ?", firstBlockHeight, args.End).Order("height").Find(&blocks)
	if res.Error != nil {
		return nil, res.Error
	}

	ret.Blocks = make([]types.BlockInfo, 0, len(blocks))
	for _, block := range blocks {
		blkInfo := types.BlockInfo{
			BlockId:   block.BlockId,
			Timestamp: block.Timestamp,
			L1Head:    uint64(block.L1Head),
			Height:    block.Height,
		}
		if len(ret.Blocks) > 0 && blkInfo.Height-1 != ret.Blocks[len(ret.Blocks)-1].Height {
			return nil, fmt.Errorf("queried blocks aren't consecutive")
		}
		ret.Blocks = append(ret.Blocks, blkInfo)
	}
	var next DBBlock
	res = oa.db.Where("timestamp >= ?", args.End).Order("height").First(&next)
	if res.Error == nil {
		ret.Next = types.BlockInfo{
			BlockId:   next.BlockId,
			Timestamp: next.Timestamp,
			L1Head:    uint64(next.L1Head),
			Height:    next.Height,
		}
	} else {
		ret.Next = types.BlockInfo{}
	}

	return ret, nil
}

func (oa *ORMArchiver) GetBlockHeadersAfterTimestamp(args *types.GetBlockHeadersByStartArgs) (*types.BlockHeadersResponse, error) {
	ret := new(types.BlockHeadersResponse)

	var startBlock DBBlock
	tx := oa.db.Where("timestamp >= ?", args.Start).Order("timestamp").First(&startBlock)
	if tx.Error != nil {
		return nil, tx.Error
	}

	firstBlockHeight := startBlock.Height

	ret.Prev = types.BlockInfo{}
	if firstBlockHeight > 1 {
		var prevBlock DBBlock
		tx := oa.db.Where("height = ?", firstBlockHeight-1).Find(&prevBlock)
		if tx.Error != nil {
			return nil, tx.Error
		}

		ret.Prev = types.BlockInfo{
			BlockId:   prevBlock.BlockId,
			Timestamp: prevBlock.Timestamp,
			L1Head:    uint64(prevBlock.L1Head),
			Height:    prevBlock.Height,
		}
	}

	var blocks []DBBlock
	res := oa.db.Where("height >= ? AND timestamp < ?", firstBlockHeight, args.End).Order("height").Find(&blocks)
	if res.Error != nil {
		return nil, res.Error
	}

	ret.Blocks = make([]types.BlockInfo, 0, len(blocks))
	for _, block := range blocks {
		blkInfo := types.BlockInfo{
			BlockId:   block.BlockId,
			Timestamp: block.Timestamp,
			L1Head:    uint64(block.L1Head),
			Height:    block.Height,
		}
		if len(ret.Blocks) > 0 && blkInfo.Height-1 != ret.Blocks[len(ret.Blocks)-1].Height {
			return nil, fmt.Errorf("queried blocks aren't consecutive")
		}
		ret.Blocks = append(ret.Blocks, blkInfo)
	}
	var next DBBlock
	res = oa.db.Where("timestamp >= ?", args.End).Order("height").First(&next)
	if res.Error == nil {
		ret.Next = types.BlockInfo{
			BlockId:   next.BlockId,
			Timestamp: next.Timestamp,
			L1Head:    uint64(next.L1Head),
			Height:    next.Height,
		}
	} else {
		ret.Next = types.BlockInfo{}
	}

	return ret, nil
}

func (oa *ORMArchiver) GetCommitmentBlocks(args *types.GetBlockCommitmentArgs, parser chain.Parser) (*types.SequencerWarpBlockResponse, error) {
	ret := new(types.SequencerWarpBlockResponse)

	if args.First < 1 {
		return nil, fmt.Errorf("the first block height is smaller than 1")
	}

	var blocks []DBBlock
	res := oa.db.Where("height >= ? AND height <= ?", args.First, args.CurrentHeight).Order("height").Limit(args.MaxBlocks).Find(&blocks)
	if res.Error != nil {
		return nil, res.Error
	}

	ret.Blocks = make([]types.SequencerWarpBlock, 0)
	for _, dbBlock := range blocks {
		blk, err := chain.UnmarshalBlock(dbBlock.Bytes, parser)
		if err != nil {
			return nil, err
		}
		blkID, err := blk.ID()
		if err != nil {
			return nil, err
		}

		header := &types.Header{
			Height:    dbBlock.Height,
			Timestamp: uint64(dbBlock.Timestamp),
			L1Head:    uint64(dbBlock.L1Head),
			TransactionsRoot: types.NmtRoot{
				Root: blkID[:],
			},
		}

		comm := header.Commit()

		parentRoot := types.NewU256().SetBytes(blk.Prnt)
		bigParentRoot := parentRoot.Int

		ret.Blocks = append(ret.Blocks, types.SequencerWarpBlock{
			BlockId:    dbBlock.BlockId,
			Timestamp:  dbBlock.Timestamp,
			L1Head:     uint64(dbBlock.L1Head),
			Height:     big.NewInt(int64(dbBlock.Height)),
			BlockRoot:  &comm.Uint256().Int,
			ParentRoot: &bigParentRoot,
		})
	}

	return ret, nil
}
