package main

import (
	"fmt"
	"log"
	"oberlblockchain/block"
	"oberlblockchain/wallet"
)

func init() {
	log.SetPrefix("Blockchain: ")
}

func main() {

	walletSwk := wallet.NewWallet()   //孙悟空
	walletZbj := wallet.NewWallet()   //猪八戒
	walletOberl := wallet.NewWallet() //矿工

	fmt.Printf("孙悟空的account:%s\n", walletSwk.BlockchainAddress())
	fmt.Printf("猪八戒的account:%s\n", walletZbj.BlockchainAddress())
	fmt.Printf("矿工  的account:%s\n", walletOberl.BlockchainAddress())

	blockchain := block.NewBlockchain(walletOberl.BlockchainAddress(), 5000)
	blockchain.Mining()
	blockchain.Print()
	//钱包 提交一笔交易
	t := wallet.NewTransaction(
		walletOberl.PrivateKey(),
		walletOberl.PublicKey(),
		walletOberl.BlockchainAddress(),
		walletZbj.BlockchainAddress(),
		8)

	//区块链 打包交易
	isAdded := blockchain.AddTransaction(
		walletOberl.BlockchainAddress(),
		walletZbj.BlockchainAddress(),
		8,
		walletOberl.PublicKey(),
		t.GenerateSignature())

	fmt.Println("这笔交易验证通过吗? ", isAdded)

	t2 := wallet.NewTransaction(
		walletSwk.PrivateKey(),
		walletSwk.PublicKey(),
		walletSwk.BlockchainAddress(),
		walletZbj.BlockchainAddress(),
		80)

	//区块链 打包交易
	isAdded = blockchain.AddTransaction(
		walletSwk.BlockchainAddress(),
		walletZbj.BlockchainAddress(),
		80,
		walletSwk.PublicKey(),
		t2.GenerateSignature())

	fmt.Println("这笔交易验证通过吗? ", isAdded)

	blockchain.Mining()
	blockchain.Print()

	fmt.Printf("孙悟空 %d\n", blockchain.CalculateTotalAmount(walletSwk.BlockchainAddress()))
	fmt.Printf("猪八戒 %d\n", blockchain.CalculateTotalAmount(walletZbj.BlockchainAddress()))
	fmt.Printf("矿工   %d\n", blockchain.CalculateTotalAmount(walletOberl.BlockchainAddress()))

}
