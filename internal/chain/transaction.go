package chain

import (
	"math/big"
	"strconv"

	"cess-bucket/configs"
	. "cess-bucket/internal/logger"
	"cess-bucket/tools"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// Storage Miner Registration Function
func Register(api *gsrpc.SubstrateAPI, imcodeAcc, ipAddr string, pledgeTokens uint64) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()

	var (
		err         error
		txhash      string
		accountInfo types.AccountInfo
	)
	if api == nil {
		api, err = GetRpcClient_Safe(configs.C.RpcAddr)
		defer Free()
		if err != nil {
			return txhash, errors.Wrap(err, "[GetRpcClient_Safe]")
		}
	}

	meta, err := GetMetadata(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetMetadataLatest]")
	}

	b, err := tools.DecodeToCessPub(imcodeAcc)
	if err != nil {
		return txhash, errors.Wrap(err, "[DecodeToCessPub]")
	}

	pTokens := strconv.FormatUint(pledgeTokens, 10)
	pTokens += configs.TokenAccuracy
	realTokens, ok := new(big.Int).SetString(pTokens, 10)
	if !ok {
		return txhash, errors.New("[big.Int.SetString]")
	}

	c, err := types.NewCall(
		meta,
		ChainTx_Sminer_Register,
		types.NewAccountID(b),
		types.Bytes([]byte(ipAddr)),
		types.NewU128(*realTokens),
	)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewExtrinsic]")
	}

	genesisHash, err := GetGenesisHash(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetGenesisHash]")
	}

	rv, err := GetRuntimeVersion(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRuntimeVersion]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", configs.PublicKey)
	if err != nil {
		return txhash, errors.Wrap(err, "[CreateStorageKey System Account]")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetStorageLatest]")
	}

	if !ok {
		return txhash, errors.New(ERR_Empty)
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	kring, err := GetKeyring(configs.C.SignatureAcc)
	if err != nil {
		return txhash, errors.Wrap(err, "GetKeyring")
	}

	// Sign the transaction
	err = ext.Sign(kring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	timeout := time.After(configs.TimeToWaitEvents)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				txhash, _ = types.EncodeToHexString(status.AsInBlock)
				keye, err := GetKeyEvents()
				if err != nil {
					return txhash, errors.Wrap(err, "GetKeyEvents")
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "GetStorageRaw")
				}

				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Out.Sugar().Infof("[%v]Decode event err:%v", txhash, err)
				}

				if len(events.Sminer_Registered) > 0 {
					for i := 0; i < len(events.Sminer_Registered); i++ {
						if string(events.Sminer_Registered[i].Acc[:]) == string(configs.PublicKey) {
							return txhash, nil
						}
					}
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return txhash, errors.Wrap(err, "<-sub")
		case <-timeout:
			return txhash, errors.New(ERR_Timeout)
		}
	}
}

// Storage miners increase deposit function
func Increase(api *gsrpc.SubstrateAPI, identifyAccountPhrase, TransactionName string, tokens *big.Int) (string, error) {

	defer func() {
		if err := recover(); err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()

	var (
		txhash      string
		accountInfo types.AccountInfo
	)

	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return txhash, errors.Wrap(err, "KeyringPairFromSecret")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return txhash, errors.Wrap(err, "GetMetadataLatest")
	}

	c, err := types.NewCall(meta, TransactionName, types.NewUCompact(tokens))
	if err != nil {
		return txhash, errors.Wrap(err, "NewCall")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "NewExtrinsic")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return txhash, errors.Wrap(err, "GetBlockHash")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return txhash, errors.Wrap(err, "GetRuntimeVersionLatest")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return txhash, errors.Wrap(err, "CreateStorageKey")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "GetStorageLatest")
	}
	if !ok {
		return txhash, errors.New("GetStorageLatest return value is empty")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "Sign")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "SubmitAndWatchExtrinsic")
	}
	defer sub.Unsubscribe()

	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				txhash, _ = types.EncodeToHexString(status.AsInBlock)
				return txhash, nil
			}
		case err = <-sub.Err():
			return txhash, err
		case <-timeout:
			return txhash, errors.New("timeout")
		}
	}
}

// Storage miner exits the mining function
func ExitMining(api *gsrpc.SubstrateAPI, identifyAccountPhrase, TransactionName string) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	var (
		txhash      string
		accountInfo types.AccountInfo
	)
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return txhash, errors.Wrap(err, "KeyringPairFromSecret")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return txhash, errors.Wrap(err, "GetMetadataLatest")
	}

	c, err := types.NewCall(meta, TransactionName)
	if err != nil {
		return txhash, errors.Wrap(err, "NewCall")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "NewExtrinsic")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return txhash, errors.Wrap(err, "GetBlockHash")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return txhash, errors.Wrap(err, "GetRuntimeVersionLatest")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return txhash, errors.Wrap(err, "CreateStorageKey")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return txhash, errors.New("GetStorageLatest return value is empty")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "Sign")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "SubmitAndWatchExtrinsic")
	}
	defer sub.Unsubscribe()

	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				txhash, _ = types.EncodeToHexString(status.AsInBlock)
				return txhash, nil
			}
		case err = <-sub.Err():
			return "", err
		case <-timeout:
			return "", errors.New("timeout")
		}
	}
}

// Storage miner redemption deposit function
func Withdraw(api *gsrpc.SubstrateAPI, identifyAccountPhrase, TransactionName string) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	var (
		txhash      string
		accountInfo types.AccountInfo
	)
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return txhash, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return txhash, errors.Wrap(err, "GetMetadataLatest err")
	}

	c, err := types.NewCall(meta, TransactionName)
	if err != nil {
		return txhash, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return txhash, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return txhash, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return txhash, errors.Wrap(err, "CreateStorageKey err")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return txhash, errors.New("GetStorageLatest return value is empty")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				txhash, _ = types.EncodeToHexString(status.AsInBlock)
				return txhash, nil
			}
		case err = <-sub.Err():
			return txhash, err
		case <-timeout:
			return txhash, errors.New("timeout")
		}
	}
}

// Bulk submission proof
func SubmitProofs(data []ProveInfo) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	var (
		txhash      string
		accountInfo types.AccountInfo
	)

	api, err := GetRpcClient_Safe(configs.C.RpcAddr)
	defer Free()
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRpcClient_Safe]")
	}

	meta, err := GetMetadata(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetMetadataLatest]")
	}

	c, err := types.NewCall(meta, SegmentBook_SubmitProve, data)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewExtrinsic]")
	}

	genesisHash, err := GetGenesisHash(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetGenesisHash]")
	}

	rv, err := GetRuntimeVersion(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRuntimeVersion]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", configs.PublicKey)
	if err != nil {
		return txhash, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetStorageLatest]")
	}

	if !ok {
		return txhash, errors.New(ERR_Empty)
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	kring, err := GetKeyring(configs.C.SignatureAcc)
	if err != nil {
		return txhash, errors.Wrap(err, "GetKeyring")
	}

	// Sign the transaction
	err = ext.Sign(kring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	timeout := time.After(configs.TimeToWaitEvents)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				txhash, _ = types.EncodeToHexString(status.AsInBlock)
				keye, err := GetKeyEvents()
				if err != nil {
					return txhash, errors.Wrap(err, "GetKeyEvents")
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "GetStorageRaw")
				}

				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Out.Sugar().Infof("[%v]Decode event err:%v", txhash, err)
				}

				if len(events.SegmentBook_ChallengeProof) > 0 && len(data) > 0 {
					if events.SegmentBook_ChallengeProof[0].Miner == data[0].MinerAcc {
						return txhash, nil
					}
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return txhash, errors.Wrap(err, "<-sub")
		case <-timeout:
			return txhash, errors.New(ERR_Timeout)
		}
	}
}

// Clear invalid files
func ClearInvalidFiles(fid types.Bytes) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	var (
		txhash      string
		accountInfo types.AccountInfo
	)
	api, err := GetRpcClient_Safe(configs.C.RpcAddr)
	defer Free()
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRpcClient_Safe]")
	}

	meta, err := GetMetadata(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetMetadataLatest]")
	}

	c, err := types.NewCall(meta, FileBank_ClearInvalidFile, fid)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewExtrinsic]")
	}

	genesisHash, err := GetGenesisHash(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetGenesisHash]")
	}

	rv, err := GetRuntimeVersion(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRuntimeVersion]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", configs.PublicKey)
	if err != nil {
		return txhash, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return txhash, errors.New(ERR_Empty)
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	kring, err := GetKeyring(configs.C.SignatureAcc)
	if err != nil {
		return txhash, errors.Wrap(err, "GetKeyring")
	}

	// Sign the transaction
	err = ext.Sign(kring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	timeout := time.After(configs.TimeToWaitEvents)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				txhash, _ = types.EncodeToHexString(status.AsInBlock)
				keye, err := GetKeyEvents()
				if err != nil {
					return txhash, errors.Wrap(err, "GetKeyEvents")
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "GetStorageRaw")
				}

				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Out.Sugar().Infof("[%v]Decode event err:%v", txhash, err)
				}

				if len(events.FileBank_ClearInvalidFile) > 0 {
					for i := 0; i < len(events.FileBank_ClearInvalidFile); i++ {
						if string(events.FileBank_ClearInvalidFile[i].Acc[:]) == string(configs.PublicKey) {
							return txhash, nil
						}
					}
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return txhash, errors.Wrap(err, "<-sub")
		case <-timeout:
			return txhash, errors.New(ERR_Timeout)
		}
	}
}

// Clear all filler files
func ClearFiller(api *gsrpc.SubstrateAPI, signaturePrk string) (int, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var accountInfo types.AccountInfo

	keyring, err := signature.KeyringPairFromSecret(signaturePrk, 0)
	if err != nil {
		return configs.Code_400, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	c, err := types.NewCall(meta, FileBank_ClearFiller)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[NewExtrinsic]")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetBlockHash]")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetRuntimeVersionLatest]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[CreateStorageKey System Account]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return configs.Code_500, errors.New("[GetStorageLatest return value is empty]")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				return configs.Code_200, nil
			}
		case err = <-sub.Err():
			return configs.Code_500, err
		case <-timeout:
			return configs.Code_500, errors.New("Timeout")
		}
	}
}

func UpdateAddress(transactionPrK, addr string) (string, int, error) {
	var (
		err         error
		accountInfo types.AccountInfo
	)

	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	api, err := NewRpcClient(configs.C.RpcAddr)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "NewRpcClient err")
	}

	keyring, err := signature.KeyringPairFromSecret(transactionPrK, 0)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "GetMetadataLatest err")
	}

	c, err := types.NewCall(meta, ChainTx_Sminer_UpdateIp, types.Bytes([]byte(addr)))
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "CreateStorageKey System  Account err")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return "", configs.Code_500, errors.New("GetStorageLatest return value is empty")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				txhash, _ := types.EncodeToHexString(status.AsInBlock)
				return txhash, configs.Code_600, nil
			}
		case err = <-sub.Err():
			return "", configs.Code_500, err
		case <-timeout:
			return "", configs.Code_500, errors.Errorf("timeout")
		}
	}
}

func UpdateIncome(transactionPrK string, acc types.AccountID) (string, int, error) {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	api, err := NewRpcClient(configs.C.RpcAddr)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "NewRpcClient err")
	}
	keyring, err := signature.KeyringPairFromSecret(transactionPrK, 0)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "GetMetadataLatest err")
	}

	c, err := types.NewCall(meta, ChainTx_Sminer_UpdateBeneficiary, acc)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "CreateStorageKey System  Account err")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return "", configs.Code_500, errors.New("GetStorageLatest return value is empty")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				txhash, _ := types.EncodeToHexString(status.AsInBlock)
				return txhash, configs.Code_600, nil
			}
		case err = <-sub.Err():
			return "", configs.Code_500, err
		case <-timeout:
			return "", configs.Code_500, errors.Errorf("timeout")
		}
	}
}
