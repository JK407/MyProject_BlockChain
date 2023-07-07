package main

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"log"
	"net/http"
	"oberlblockchain/block"
	"oberlblockchain/utils"
	"oberlblockchain/wallet"
	"strconv"
	"sync"
)

//var cache map[string]*block.Blockchain = make(map[string]*block.Blockchain)

var DB *sql.DB

var mutex sync.Mutex

type BlockchainServer struct {
	port  uint16
	cache map[string]*block.Blockchain
}

func NewBlockchainServer(port uint16) *BlockchainServer {
	InitDB()
	return &BlockchainServer{
		port: port,
	}
}

func (bcs *BlockchainServer) Port() uint16 {
	return bcs.port
}
func InitDB() {
	db, err := sql.Open("mysql", "root:root@tcp(localhost:3306)/blockchain")
	if err != nil {
		log.Fatal(err)
	}

	DB = db
}


func (bcs *BlockchainServer) GetBlockchain() *block.Blockchain {
	bc := bcs.getCache("blockchain")
	//fmt.Println(bc)
	if bc == nil {
		// 从数据库中初始化区块链
		//fmt.Println("bc == nil")
		bc, err := bcs.InitBlockchain()
		if err != nil {
			log.Printf("错误: 初始化区块链时出错: %v", err)
			return nil
		}
		return bc

	}
	//从数据库中加载区块链数据并返回
	err := bcs.AppendBlockchainToDB(bc)
	if err != nil {
		log.Printf("错误: 将区块链追加到数据库时出错: %v", err)
		return nil
	}
	bcs.setCache("blockchain",bc)

	return bc
}

func (bcs *BlockchainServer) AppendBlockchainToDB(bc *block.Blockchain) error {
	// 获取数据库中已存在的区块数量
	var existingBlocksCount int
	err := DB.QueryRow("SELECT COUNT(*) FROM blocks").Scan(&existingBlocksCount)
	if err != nil {
		return fmt.Errorf("查询数据库中已存在的区块数量时出错: %v", err)
	}

	// 检查区块链中的区块数量是否大于数据库中已存在的区块数量
	if len(bc.Chain()) <= existingBlocksCount {
		// 区块链中的区块数量不大于数据库中已存在的区块数量，无需追加数据到数据库
		return nil
	}

	// 遍历区块链中的新增区块，并将其插入到数据库中
	stmt, err := DB.Prepare("INSERT INTO blocks (nonce, previous_hash, timestamp, transactions) VALUES (?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("准备插入区块的SQL语句时出错: %v", err)
	}

	for _, b := range bc.Chain()[existingBlocksCount:] {
		// 转换 previousHash 为字符串类型
		previousHashStr := fmt.Sprintf("%x", b.PreviousHash[:])

		// 序列化交易数据
		transactionsData, err := json.Marshal(b.Transactions)
		if err != nil {
			return fmt.Errorf("序列化交易数据时出错: %v", err)
		}

		// 插入区块数据到数据库
		_, err = stmt.Exec(b.Nonce, previousHashStr, b.Timestamp, transactionsData)
		if err != nil {
			return fmt.Errorf("将区块插入数据库时出错: %v", err)
		}
	}

	return nil
}

func (bcs *BlockchainServer) InitBlockchain() (*block.Blockchain, error) {
	// 从数据库中获取区块链数据
	rows, err := DB.Query("SELECT nonce, previous_hash, timestamp, transactions FROM blocks")
	if err != nil {
		return nil, fmt.Errorf("查询数据库时出错: %v", err)
	}
	defer rows.Close()

	// 创建一个新的区块链对象
	minersWallet := wallet.NewWallet()
	newBlockchain := block.NewBlockchain(minersWallet.BlockchainAddress(), bcs.Port())
	color.Magenta("===矿工帐号信息====\n")
	color.Magenta("矿工private_key\n %v\n", minersWallet.PrivateKeyStr())
	color.Magenta("矿工publick_key\n %v\n", minersWallet.PublicKeyStr())
	color.Magenta("矿工blockchain_address\n %s\n", minersWallet.BlockchainAddress())
	color.Magenta("===============\n")

	for rows.Next() {
		var nonce int
		var previousHashStr string
		var timestamp int64
		var transactionsData string
		err := rows.Scan(&nonce, &previousHashStr, &timestamp, &transactionsData)
		if err != nil {
			return nil, fmt.Errorf("扫描数据库行时出错: %v", err)
		}

		// 解析交易数据
		var transactions []*block.Transaction
		err = json.Unmarshal([]byte(transactionsData), &transactions)
		if err != nil {
			return nil, fmt.Errorf("解析交易数据时出错: %v", err)
		}
		var previousHashBytes [32]byte
		_, err = hex.Decode(previousHashBytes[:], []byte(previousHashStr))

		// 创建区块对象并添加到区块链中
		b := &block.Block{
			Nonce:        nonce,
			PreviousHash: previousHashBytes,
			Timestamp:    timestamp,
			Transactions: transactions,
		}
		newBlockchain.AddBlock(b)
	}

	// 更新缓存中的区块链数据
	bcs.setCache("blockchain", newBlockchain)

	return newBlockchain, nil
}

func (bcs *BlockchainServer) getCache(key string) *block.Blockchain {
	mutex.Lock()
	defer mutex.Unlock()

	if bcs.cache == nil {
		bcs.cache = make(map[string]*block.Blockchain)
	}

	bc, ok := bcs.cache[key]
	if !ok {
		return nil
	}

	return bc
}

func (bcs *BlockchainServer) setCache(key string, bc *block.Blockchain) {
	mutex.Lock()
	defer mutex.Unlock()

	if bcs.cache == nil {
		bcs.cache = make(map[string]*block.Blockchain)
	}

	bcs.cache[key] = bc
}

func (bcs *BlockchainServer) GetChain(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		bc := bcs.GetBlockchain()
		m, _ := bc.MarshalJSON()
		io.WriteString(w, string(m[:]))
	default:
		log.Printf("ERROR: Invalid HTTP Method")

	}
}

func (bcs *BlockchainServer) Transactions(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		{
			// Get:显示交易池的内容，Mine成功后清空交易池
			w.Header().Add("Content-Type", "application/json")
			bc := bcs.GetBlockchain()

			transactions := bc.TransactionPool()
			m, _ := json.Marshal(struct {
				Transactions []*block.Transaction `json:"transactions"`
				Length       int                  `json:"length"`
			}{
				Transactions: transactions,
				Length:       len(transactions),
			})
			io.WriteString(w, string(m[:]))
		}
	case http.MethodPost:
		{
			log.Printf("\n\n\n")
			log.Println("接受到wallet发送的交易")
			decoder := json.NewDecoder(req.Body)
			var t block.TransactionRequest
			err := decoder.Decode(&t)
			if err != nil {
				log.Printf("ERROR: %v", err)
				io.WriteString(w, string(utils.JsonStatus("Decode Transaction失败")))
				return
			}

			log.Println("发送人公钥SenderPublicKey:", *t.SenderPublicKey)
			log.Println("发送人私钥SenderPrivateKey:", *t.SenderBlockchainAddress)
			log.Println("接收人地址RecipientBlockchainAddress:", *t.RecipientBlockchainAddress)
			log.Println("金额Value:", *t.Value)
			log.Println("交易Signature:", *t.Signature)

			if !t.Validate() {
				log.Println("ERROR: missing field(s)")
				io.WriteString(w, string(utils.JsonStatus("fail")))
				return
			}

			publicKey := utils.PublicKeyFromString(*t.SenderPublicKey)
			signature := utils.SignatureFromString(*t.Signature)
			bc := bcs.GetBlockchain()

			isCreated := bc.CreateTransaction(*t.SenderBlockchainAddress,
				*t.RecipientBlockchainAddress, uint64(*t.Value), publicKey, signature)

			w.Header().Add("Content-Type", "application/json")
			var m []byte
			if !isCreated {
				w.WriteHeader(http.StatusBadRequest)
				m = utils.JsonStatus("fail[from:blockchainServer]")
			} else {
				w.WriteHeader(http.StatusCreated)
				m = utils.JsonStatus("success[from:blockchainServer]")
			}
			io.WriteString(w, string(m))

		}
	case http.MethodPut:
		// PUT方法 用于在另据节点同步交易
		decoder := json.NewDecoder(req.Body)
		var t block.TransactionRequest
		err := decoder.Decode(&t)
		if err != nil {
			log.Printf("ERROR: %v", err)
			io.WriteString(w, string(utils.JsonStatus("fail")))
			return
		}
		if !t.Validate() {
			log.Println("ERROR: missing field(s)")
			io.WriteString(w, string(utils.JsonStatus("fail")))
			return
		}
		publicKey := utils.PublicKeyFromString(*t.SenderPublicKey)
		signature := utils.SignatureFromString(*t.Signature)
		bc := bcs.GetBlockchain()

		isUpdated := bc.AddTransaction(*t.SenderBlockchainAddress,
			*t.RecipientBlockchainAddress, int64(*t.Value), publicKey, signature)

		w.Header().Add("Content-Type", "application/json")
		var m []byte
		if !isUpdated {
			w.WriteHeader(http.StatusBadRequest)
			m = utils.JsonStatus("fail")
		} else {
			m = utils.JsonStatus("success")
		}
		io.WriteString(w, string(m))
	case http.MethodDelete:
		bc := bcs.GetBlockchain()
		bc.ClearTransactionPool()
		io.WriteString(w, string(utils.JsonStatus("success")))
	default:
		log.Println("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) Mine(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		bc := bcs.GetBlockchain()
		isMined := bc.Mining()

		var m []byte
		if !isMined {
			w.WriteHeader(http.StatusBadRequest)
			m = utils.JsonStatus("挖矿失败[from:Mine]")
		} else {
			m = utils.JsonStatus("挖矿成功[from:Mine]")
		}
		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m))
	default:
		log.Println("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) StartMine(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		bc := bcs.GetBlockchain()
		bc.StartMining()

		m := utils.JsonStatus("success")
		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m))
	default:
		log.Println("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) Amount(w http.ResponseWriter, req *http.Request) {

	switch req.Method {
	case http.MethodPost:

		var data map[string]interface{}
		// 解析JSON数据

		err := json.NewDecoder(req.Body).Decode(&data)
		if err != nil {
			http.Error(w, "无法解析JSON数据", http.StatusBadRequest)
			return
		}
		// 获取JSON字段的值
		blockchainAddress := data["blockchain_address"].(string)

		color.Green("查询账户: %s 余额请求", blockchainAddress)

		amount := bcs.GetBlockchain().CalculateTotalAmount(blockchainAddress)

		ar := &block.AmountResponse{Amount: amount}
		m, _ := ar.MarshalJSON()

		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m[:]))

	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) Consensus(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPut:
		color.Cyan("####################Consensus###############")
		bc := bcs.GetBlockchain()
		replaced := bc.ResolveConflicts()
		color.Red("[共识]Consensus replaced :%v\n", replaced)
		w.Header().Add("Content-Type", "application/json")
		if replaced {
			io.WriteString(w, string(utils.JsonStatus("success")))
		} else {
			io.WriteString(w, string(utils.JsonStatus("fail")))
		}
	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) ListTransactionsByAddress(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		var data map[string]interface{}
		// 解析JSON数据

		err := json.NewDecoder(req.Body).Decode(&data)
		if err != nil {
			http.Error(w, "无法解析JSON数据", http.StatusBadRequest)
			return
		}
		// 获取JSON字段的值
		blockchainAddress := data["blockchain_address"].(string)

		color.Green("查询账户: %s 交易请求", blockchainAddress)

		// 调用BlockchainServer的方法获取与地址相关的交易
		bc := bcs.GetBlockchain()

		// 创建用于存储符合条件的交易的切片
		var transactions []*block.Transaction

		// 遍历区块链中的每个区块
		for _, block := range bc.Chain() {
			// 遍历当前区块中的每个交易
			for _, transaction := range block.GetTransactions() {
				// 检查发送方和接收方的地址是否匹配指定地址
				if transaction.SenderAddress == blockchainAddress || transaction.ReceiveAddress == blockchainAddress {
					// 将符合条件的交易添加到切片中
					transactions = append(transactions, transaction)
				}
			}
		}
		// 将交易切片转换为 JSON 格式
		transactionJSON, err := json.Marshal(transactions)
		if err != nil {
			log.Printf("ERROR: Failed to marshal transactions")
			http.Error(w, "Failed to marshal transactions", http.StatusInternalServerError)
			return
		}
		// 设置响应头并返回交易列表
		w.Header().Set("Content-Type", "application/json")
		w.Write(transactionJSON)
	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) GetBlockByNum(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		//从URL查询参数中获取名为index的值，该值表示要查询的区块的索引
		index := req.URL.Query().Get("index")
		blockchain := bcs.GetBlockchain()
		//将index参数转换为整数类型
		blockIndex, err := strconv.Atoi(index)
		if err != nil {
			log.Printf("ERROR: Invalid index")
			http.Error(w, "Invalid index", http.StatusBadRequest)
			return
		}

		if blockIndex < 0 || blockIndex >= len(blockchain.Chain()) {
			log.Printf("ERROR: Block not found")
			http.Error(w, "Block not found", http.StatusNotFound)
			return
		}

		block := blockchain.Chain()[blockIndex]
		blockJSON, err := json.Marshal(block)
		if err != nil {
			log.Printf("ERROR: Failed to marshal block")
			http.Error(w, "Failed to marshal block", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(blockJSON)

	default:
		log.Printf("ERROR: Invalid HTTP Method")
	}
}

func (bcs *BlockchainServer) GetBlockByHash(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		var data map[string]interface{}
		// 解析JSON数据

		err := json.NewDecoder(req.Body).Decode(&data)
		if err != nil {
			http.Error(w, "无法解析JSON数据", http.StatusBadRequest)
			return
		}

		// 获取JSON字段的值
		hashStr := data["hash"].(string)

		// 将哈希字符串转换为字节数组
		hashBytes, err := hex.DecodeString(hashStr)
		if err != nil {
			http.Error(w, "无效的哈希值", http.StatusBadRequest)
			return
		}

		// 将字节数组转换为 [32]byte 类型
		var hash [32]byte
		copy(hash[:], hashBytes)

		color.Green("查询区块 %x", hash)

		bc := bcs.GetBlockchain()
		var blocks []*block.Block

		// 遍历区块链中的每个区块
		for _, blk := range bc.Chain() {
			// 检查区块的哈希值是否与指定哈希值相同
			if blk.GetBlockHash() == hash {
				// 将符合条件的区块添加到切片中
				blocks = append(blocks, blk)
			}
		}

		// 将区块切片转换为 JSON 格式
		blockJSON, err := json.Marshal(blocks)
		if err != nil {
			log.Printf("ERROR: Failed to marshal blocks")
			http.Error(w, "Failed to marshal blocks", http.StatusInternalServerError)
			return
		}

		// 设置响应头并返回区块列表
		w.Header().Set("Content-Type", "application/json")
		w.Write(blockJSON)

	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}
func (bcs *BlockchainServer) ListTransactionsByTxHash(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		var data map[string]interface{}
		// 解析JSON数据
		err := json.NewDecoder(req.Body).Decode(&data)
		if err != nil {
			http.Error(w, "无法解析JSON数据", http.StatusBadRequest)
			return
		}

		// 获取JSON字段的值
		hashStr := data["transaction_hash"].(string)

		// 将哈希字符串转换为字节数组
		hashBytes, err := hex.DecodeString(hashStr)
		if err != nil {
			http.Error(w, "无效的哈希值", http.StatusBadRequest)
			return
		}

		// 将字节数组转换为 [32]byte 类型
		var hash [32]byte
		copy(hash[:], hashBytes)

		color.Green("查询交易 %x", hash)

		bc := bcs.GetBlockchain()
		var transactions []*block.Transaction

		// 遍历区块链中的每个区块
		for _, blk := range bc.Chain() {
			// 遍历区块中的每个交易
			for _, tx := range blk.GetTransactions() {
				// 检查交易的哈希值是否与指定哈希值相同
				if tx.TxHash == hash {
					// 将符合条件的交易添加到切片中
					transactions = append(transactions, tx)
				}
			}
		}

		// 将交易切片转换为 JSON 格式
		transactionJSON, err := json.Marshal(transactions)
		if err != nil {
			log.Printf("ERROR: Failed to marshal transactions")
			http.Error(w, "Failed to marshal transactions", http.StatusInternalServerError)
			return
		}

		// 设置响应头并返回交易列表
		w.Header().Set("Content-Type", "application/json")
		w.Write(transactionJSON)

	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) Run() {
	bcs.GetBlockchain().Run()

	http.HandleFunc("/", bcs.GetChain)

	http.HandleFunc("/transactions", bcs.Transactions) //GET 方式和  POST方式
	http.HandleFunc("/mine", bcs.Mine)
	http.HandleFunc("/mine/start", bcs.StartMine)
	http.HandleFunc("/amount", bcs.Amount)
	http.HandleFunc("/consensus", bcs.Consensus)

	http.HandleFunc("/ListTransactionsByTxHash", bcs.ListTransactionsByTxHash)
	http.HandleFunc("/ListTransactions", bcs.ListTransactionsByAddress)
	http.HandleFunc("/GetBlockByNum", bcs.GetBlockByNum)
	http.HandleFunc("/GetBlockByHash", bcs.GetBlockByHash)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(int(bcs.Port())), nil))

}
