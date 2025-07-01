package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoclient"
	rpcclient "github.com/gnolang/gno/tm2/pkg/bft/rpc/client"
	"github.com/pelletier/go-toml/v2"
	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"

	"github.com/gnoverse/gno-psql-indexer/internal/db"
)

var (
	// TODO: Parse genesis file to get moniker
	validatorNameAddr = map[string]string{}
	validatorAddrName = map[string]string{}
)

var (
	configFilePath = flag.String("config", "config.toml", "Path to the configuration file")
)

type Config struct {
	RPC struct {
		Endpoint string `toml:"endpoint"`
	} `toml:"rpc"`

	DB struct {
		Endpoint string `toml:"endpoint"`
	} `toml:"database"`

	Chain struct {
		Validators map[string]string `toml:"validators"`
	} `toml:"chain"`

	Scrapper struct {
		BatchWrite      int `toml:"batch_write"`
		GoroBlockParser int `toml:"goro_block_parser"`

		BufferChBlocks  int `toml:"buffer_chan_blocks"`
		BufferChHeights int `toml:"buffer_chan_heights"`
	} `toml:"scrapper"`
}

func ParseConfig(filePath string) (*Config, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	config := Config{}

	err = toml.NewDecoder(f).Decode(&config)
	return &config, err
}

func main() {
	flag.Parse()

	config, err := ParseConfig(*configFilePath)
	if err != nil {
		logrus.WithError(err).Fatal()
	}

	validatorNameAddr = config.Chain.Validators

	rpcClient, err := rpcclient.NewHTTPClient(config.RPC.Endpoint)
	if err != nil {
		logrus.WithError(err).Fatal()
	}

	rpc := gnoclient.Client{
		RPCClient: rpcClient,
	}

	// Initialize database
	dbclient, err := db.NewDB(config.DB.Endpoint)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to connect to DB")
	}

	err = dbclient.InitTables()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create tables")
	}

	// Get blockchain latest height
	latestBlock, err := rpc.LatestBlockHeight()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to get latest block height")
	}

	// Get latest height stored
	latestBlockHeightStored, err := dbclient.GetLatestBlockHeight()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to get latest block height from db")
	}

	logrus.WithFields(logrus.Fields{
		"latest_block":        latestBlock,
		"latest_stored_block": latestBlockHeightStored,
	}).Info("Starting")

	height := latestBlockHeightStored + int64(config.Scrapper.BatchWrite)
	if height > latestBlock {
		height = latestBlock
	}

	// Insert validators
	validatorResp, err := rpc.RPCClient.Validators(&height)
	if err != nil {
		logrus.WithError(err).Fatal("failed to query validators")
	}

	validators := make([]db.Validator, len(validatorResp.Validators))
	for i, v := range validatorResp.Validators {
		addr := v.Address.Bech32().String()

		for k, v := range validatorNameAddr {
			if v == addr {
				validatorAddrName[v] = k
			}
		}

		moniker := validatorAddrName[v.Address.Bech32().String()]

		if moniker == "" {
			logrus.Errorf("Unknow validator with address: %s", addr)
			return
		}

		validators[i] = db.Validator{
			Addr: addr,
			Name: moniker,
		}
	}

	if err := dbclient.InsertValidators(validators); err != nil {
		logrus.WithError(err).Fatal()
	}

	chHeights := make(chan int64, config.Scrapper.BufferChHeights)
	chBlocks := make(chan *db.Block, config.Scrapper.BufferChBlocks)

	wgHeights := sync.WaitGroup{}
	wgBlocks := sync.WaitGroup{}

	wgBlocks.Add(1)
	go func() {
		defer wgBlocks.Done()

		blockBuff := make([]*db.Block, 0, config.Scrapper.BatchWrite)

		bar := progressbar.Default(-1)

		for block := range chBlocks {
			bar.Add(1)
			bar.Describe(fmt.Sprintf("height: %d / %d", block.Height, latestBlock))

			blockBuff = append(blockBuff, block)

			if len(blockBuff) >= config.Scrapper.BatchWrite {
				if err := dbclient.InsertBlocks(blockBuff); err != nil {
					logrus.WithError(err).Error()
				}

				blockBuff = blockBuff[:0]
			}
		}
	}()

	for i := 0; i < config.Scrapper.GoroBlockParser; i++ {
		wgHeights.Add(1)

		go func() {
			defer wgHeights.Done()

			for height := range chHeights {
				resp, err := rpc.Block(height)
				if err != nil {
					logrus.WithError(err).WithFields(logrus.Fields{
						"height": height,
					}).Error()
					continue
				}

				b, err := db.NewBlock(resp.Block, validatorAddrName)
				if err != nil {
					if strings.Contains(err.Error(), `block proposer is an unknown validator`) {
						// TODO: Get validators on this block and update database
					} else if strings.Contains(err.Error(), `missing validators informations`) {
						logrus.WithError(err).Error()
						continue
					}

					logrus.WithError(err).WithFields(logrus.Fields{
						"height": height,
					}).Error()
					continue
				}

				chBlocks <- b
			}
		}()
	}

	fromHeight := latestBlockHeightStored

	// Get all missings blocks in the database
	missingBlocks, err := dbclient.GetMissingBlocksInSeries()
	if err != nil {
		logrus.WithError(err).Fatal()
	}

	// config.Scrapper.BatchWrite = 1
	for _, height := range missingBlocks {
		chHeights <- height
	}

	retry := 0
	// Catchup quikly blocks history
	for height := fromHeight; height <= latestBlock; height++ {
		select {
		case chHeights <- height:
			retry = 0
			continue
		default:
			if retry >= 3 {
				break
			}
			time.Sleep(time.Second * 1)
			retry += 1
		}
	}

	fromHeight = latestBlock

	go func() {
		for {
			latestBlock, err := rpc.LatestBlockHeight()
			if err != nil {
				continue
			}

			for fromHeight <= latestBlock {
				chHeights <- fromHeight
				fromHeight++
			}

			if len(chHeights) < 10 {
				config.Scrapper.BatchWrite = 1
			}

			time.Sleep(time.Second * 2)
		}
	}()

	wgHeights.Wait()
	close(chBlocks)
	wgBlocks.Wait()
}
