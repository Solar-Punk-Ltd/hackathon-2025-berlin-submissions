package screens

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethersphere/bee/v2/pkg/sctx"
	"github.com/ethersphere/bee/v2/pkg/transaction"
)

type DataContractInterface interface {
	SendDataToTarget(ctx context.Context, target common.Address, owner, actRef []byte, topic string) (receipt *types.Receipt, err error)
	SubscribeDataSentToTarget(ctx context.Context, client *ethclient.Client, sink chan<- types.Log) (ethereum.Subscription, error)
}

type datacontract struct {
	owner               common.Address
	dataContractAddress common.Address
	dataContractABI     abi.ABI
	transactionService  transaction.Service
	gasLimit            uint64
	dataSentToTarget    common.Hash
}

func NewDataContract(
	owner common.Address,
	dataContractAddress common.Address,
	dataContractABI abi.ABI,
	transactionService transaction.Service,
	setGasLimit bool,
) DataContractInterface {

	var gasLimit uint64
	if setGasLimit {
		gasLimit = transaction.DefaultGasLimit
	}

	return &datacontract{
		owner:               owner,
		dataContractAddress: dataContractAddress,
		dataContractABI:     dataContractABI,
		transactionService:  transactionService,
		gasLimit:            gasLimit,
		dataSentToTarget:    dataContractABI.Events["DataSentToTarget"].ID,
	}
}

func (c *datacontract) SendDataToTarget(ctx context.Context, target common.Address, owner, actRef []byte, topic string) (receipt *types.Receipt, err error) {

	callData, err := c.dataContractABI.Pack("sendDataToTarget", target, owner, actRef, topic)
	if err != nil {
		return nil, err
	}

	receipt, err = c.sendTransaction(ctx, callData, "sendDataToTarget")
	if err != nil {
		return nil, fmt.Errorf("send data to target: %w", err)
	}

	return receipt, nil
}

func (c *datacontract) SubscribeDataSentToTarget(ctx context.Context, client *ethclient.Client, sink chan<- types.Log) (ethereum.Subscription, error) {
	if client == nil {
		return nil, errors.New("ethclient.Client is nil")
	}

	currentBlock, err := client.BlockNumber(context.Background())
	if err != nil {
		log.Println("Error getting current block number for admin subscription:", err)
	} else {
		currentBlock = 40581246
	}
	log.Printf("Obtained currentBlock: %d for admin subscription", currentBlock)

	fromBlockBigInt := new(big.Int).SetUint64(currentBlock)

	log.Printf("Subscribing to DataContract DataSentToTarget events from block %s", fromBlockBigInt.String())

	const blockPageSize = 500
	query := ethereum.FilterQuery{
		Addresses: []common.Address{c.dataContractAddress},
		Topics:    [][]common.Hash{{c.dataSentToTarget}},
		FromBlock: fromBlockBigInt,
		ToBlock:   big.NewInt(int64(currentBlock + blockPageSize - 1)),
	}

	logs := make(chan types.Log)

	sub, err := client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to DataSentToTarget events: %w", err)
	}

	go func() {
		defer close(sink)
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Subscription context cancelled, unsubscribing.")
				sub.Unsubscribe()
				return
			case err := <-sub.Err():
				fmt.Printf("Event subscription error for DataSentToTarget: %v\\n", err)
				return
			case vLog := <-logs:
				select {
				case sink <- vLog:
					fmt.Printf("Received DataSentToTarget TxHash: %s\\n", vLog.TxHash.Hex())
					fmt.Printf("Received DataSentToTarget data: %s\\n", vLog.Data)
				case <-ctx.Done():
					fmt.Println("Subscription context cancelled while trying to send log, unsubscribing.")
					sub.Unsubscribe()
					return
				}
			}
		}
	}()

	return sub, nil
}

func (c *datacontract) sendTransaction(ctx context.Context, callData []byte, desc string) (receipt *types.Receipt, err error) {
	request := &transaction.TxRequest{
		To:          &c.dataContractAddress,
		Data:        callData,
		GasPrice:    sctx.GetGasPrice(ctx),
		GasLimit:    max(sctx.GetGasLimit(ctx), c.gasLimit),
		Value:       big.NewInt(0),
		Description: desc,
	}

	defer func() {
		err = c.transactionService.UnwrapABIError(
			ctx,
			request,
			err,
			c.dataContractABI.Errors,
		)
	}()

	txHash, err := c.transactionService.Send(ctx, request, transaction.DefaultTipBoostPercent)
	if err != nil {
		return nil, err
	}

	receipt, err = c.transactionService.WaitForReceipt(ctx, txHash)
	if err != nil {
		return nil, err
	}

	if receipt.Status == 0 {
		return nil, transaction.ErrTransactionReverted
	}

	return receipt, nil
}
