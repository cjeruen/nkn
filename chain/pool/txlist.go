package pool

import (
	"errors"
	"sync"

	"github.com/nknorg/nkn/common"
	"github.com/nknorg/nkn/transaction"
	"github.com/nknorg/nkn/util/log"
)

var (
	ErrNonceSortedTxsEmpty = errors.New("Empty NonceSortedTxs")
	ErrNonceOutofRange     = errors.New("nonce is not in range")
)

type sortTxnsByNonce []*transaction.Transaction

func (s sortTxnsByNonce) Len() int      { return len(s) }
func (s sortTxnsByNonce) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortTxnsByNonce) Less(i, j int) bool {
	if s[i].UnsignedTx.Nonce > s[j].UnsignedTx.Nonce {
		return false
	} else {
		return true
	}
}

// NonceSortedTxs store the txns that can be add into blockchain.
// The txns are sorted by nonce in Increasing order.
type NonceSortedTxs struct {
	mu      sync.RWMutex
	account common.Uint160
	txs     map[common.Uint256]*transaction.Transaction // txns belong to The same address
	idx     []common.Uint256                            // the sequential tx hash list
	cap     int                                         // the capacity of the tx hash list
}

// NewNonceSortedTxs return a new NonceSortedTxs instance
func NewNonceSortedTxs(acc common.Uint160, cap int) *NonceSortedTxs {
	return &NonceSortedTxs{
		account: acc,
		txs:     make(map[common.Uint256]*transaction.Transaction),
		idx:     make([]common.Uint256, 0),
		cap:     cap,
	}
}

func (nst *NonceSortedTxs) len() int {
	return len(nst.idx)
}

func (nst *NonceSortedTxs) Len() int {
	nst.mu.RLock()
	defer nst.mu.RUnlock()
	return nst.len()
}

func (nst *NonceSortedTxs) empty() bool {
	return nst.len() == 0
}

func (nst *NonceSortedTxs) Empty() bool {
	nst.mu.RLock()
	defer nst.mu.RUnlock()
	return nst.empty()
}

func (nst *NonceSortedTxs) full() bool {
	return nst.len() >= nst.cap
}

func (nst *NonceSortedTxs) Full() bool {
	nst.mu.RLock()
	defer nst.mu.RUnlock()
	return nst.full()
}

func (nst *NonceSortedTxs) Push(tx *transaction.Transaction) error {
	nst.mu.Lock()
	defer nst.mu.Unlock()

	hash := tx.Hash()
	nst.idx = append(nst.idx, hash)
	nst.txs[hash] = tx

	return nil
}

func (nst *NonceSortedTxs) Pop() (*transaction.Transaction, error) {
	nst.mu.Lock()
	defer nst.mu.Unlock()

	if nst.empty() {
		return nil, ErrNonceSortedTxsEmpty
	}

	hash := nst.idx[0]
	nst.idx = nst.idx[1:]
	tx := nst.txs[hash]
	delete(nst.txs, hash)

	return tx, nil
}

func (nst *NonceSortedTxs) Seek() (*transaction.Transaction, error) {
	nst.mu.RLock()
	defer nst.mu.RUnlock()

	if nst.empty() {
		return nil, ErrNonceSortedTxsEmpty
	}

	return nst.txs[nst.idx[0]], nil
}

func (nst *NonceSortedTxs) getNonce(hash common.Uint256) uint64 {
	if tx, ok := nst.txs[hash]; ok {
		return tx.UnsignedTx.Nonce
	}

	panic("no such tx in NonceSortedTxs")
}

func (nst *NonceSortedTxs) Add(tx *transaction.Transaction) error {
	nst.mu.Lock()
	defer nst.mu.Unlock()

	if nst.empty() {
		return ErrNonceSortedTxsEmpty
	}

	txNonce := tx.UnsignedTx.Nonce
	if txNonce < nst.getNonce(nst.idx[0]) || txNonce > nst.getNonce(nst.idx[nst.len()-1]) {
		return ErrNonceOutofRange
	}

	origHash := nst.idx[txNonce-nst.getNonce(nst.idx[0])]
	nst.idx[txNonce-nst.getNonce(nst.idx[0])] = tx.Hash()
	delete(nst.txs, origHash)
	nst.txs[tx.Hash()] = tx

	return nil
}

func (nst *NonceSortedTxs) Get(nonce uint64) (*transaction.Transaction, error) {
	nst.mu.RLock()
	defer nst.mu.RUnlock()

	if nst.empty() {
		return nil, ErrNonceSortedTxsEmpty
	}

	if nonce < nst.getNonce(nst.idx[0]) || nonce > nst.getNonce(nst.idx[nst.len()-1]) {
		return nil, ErrNonceOutofRange
	}

	hash := nst.idx[nonce-nst.getNonce(nst.idx[0])]

	return nst.txs[hash], nil
}

func (nst *NonceSortedTxs) GetAllTransactions() []*transaction.Transaction {
	txns := make([]*transaction.Transaction, 0)
	for _, txnHash := range nst.idx {
		txns = append(txns, nst.txs[txnHash])
	}

	return txns
}

func (nst *NonceSortedTxs) GetLatestNonce() (uint64, error) {
	nst.mu.RLock()
	defer nst.mu.RUnlock()

	if nst.empty() {
		return 0, ErrNonceSortedTxsEmpty
	}

	return nst.getNonce(nst.idx[nst.len()-1]), nil

}

func (nst *NonceSortedTxs) ExistTx(hash common.Uint256) bool {
	nst.mu.RLock()
	defer nst.mu.RUnlock()

	if _, ok := nst.txs[hash]; ok {
		return true
	}

	return false
}

func (nst *NonceSortedTxs) Dump() {
	nst.mu.RLock()
	defer nst.mu.RUnlock()
	addr, _ := nst.account.ToAddress()
	log.Info("account:", addr)
	log.Info("txs:", len(nst.txs))
	for h, tx := range nst.txs {
		log.Info(h.ToHexString(), ":", tx.UnsignedTx.Nonce)
	}
	log.Info("idx:", len(nst.idx))
	for _, h := range nst.idx {
		log.Info(h.ToHexString())
	}
	log.Info("cap:", nst.cap)
}

//type FeeSortedTxns []*Transaction
