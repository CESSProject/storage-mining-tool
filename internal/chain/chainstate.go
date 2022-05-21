package chain

import (
	"cess-bucket/configs"
	. "cess-bucket/internal/logger"
	"encoding/binary"
	"fmt"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// Get miner information on the cess chain
func GetMinerItems(phrase string) (Chain_MinerItems, int, error) {
	var (
		err   error
		mdata Chain_MinerItems
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	account, err := signature.KeyringPairFromSecret(phrase, 0)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	key, err := types.CreateStorageKey(meta, State_Sminer, Sminer_MinerItems, account.PublicKey)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return mdata, configs.Code_404, nil
	}
	return mdata, configs.Code_200, nil
}

// Get miner information on the cess chain
func GetMinerDetailInfo(identifyAccountPhrase, chainModule, chainModuleMethod1, chainModuleMethod2 string) (CessChain_MinerInfo, error) {
	var (
		err   error
		mdata CessChain_MinerInfo
		m1    Chain_MinerItems
		m2    Chain_MinerDetails
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, errors.Wrap(err, "GetMetadataLatest err")
	}

	account, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return mdata, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod1, account.PublicKey)
	if err != nil {
		return mdata, errors.Wrap(err, "CreateStorageKey err")
	}

	_, err = api.RPC.State.GetStorageLatest(key, &m1)
	if err != nil {
		return mdata, errors.Wrap(err, "GetStorageLatest err")
	}

	eraIndexSerialized := make([]byte, 8)
	binary.LittleEndian.PutUint64(eraIndexSerialized, uint64(m1.Peerid))

	key, err = types.CreateStorageKey(meta, chainModule, chainModuleMethod2, types.NewBytes(eraIndexSerialized))
	if err != nil {
		return mdata, errors.Wrap(err, "CreateStorageKey err")
	}

	_, err = api.RPC.State.GetStorageLatest(key, &m2)
	if err != nil {
		return mdata, errors.Wrap(err, "GetStorageLatest err")
	}

	mdata.MinerItems.Peerid = m1.Peerid
	mdata.MinerItems.Beneficiary = m1.Beneficiary
	mdata.MinerItems.ServiceAddr = m1.ServiceAddr
	mdata.MinerItems.Collaterals = m1.Collaterals
	mdata.MinerItems.Earnings = m1.Earnings
	mdata.MinerItems.Locked = m1.Locked
	mdata.MinerItems.State = m1.State
	mdata.MinerItems.Power = m1.Power
	mdata.MinerItems.Space = m1.Space
	mdata.MinerItems.PublicKey = m1.PublicKey

	mdata.MinerDetails.Address = m2.Address
	mdata.MinerDetails.Beneficiary = m2.Beneficiary
	mdata.MinerDetails.ServiceAddr = m2.ServiceAddr
	mdata.MinerDetails.Power = m2.Power
	mdata.MinerDetails.Space = m2.Space
	mdata.MinerDetails.Total_reward = m2.Total_reward
	mdata.MinerDetails.Total_rewards_currently_available = m2.Total_rewards_currently_available
	mdata.MinerDetails.Totald_not_receive = m2.Totald_not_receive
	mdata.MinerDetails.Collaterals = m2.Collaterals

	return mdata, nil
}

// Get scheduler information on the cess chain
func GetSchedulerInfo() ([]SchedulerInfo, error) {
	var (
		err  error
		data []SchedulerInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, errors.Wrapf(err, "[%v.%v:GetMetadataLatest]", State_FileMap, FileMap_SchedulerInfo)
	}

	key, err := types.CreateStorageKey(meta, State_FileMap, FileMap_SchedulerInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "[%v.%v:CreateStorageKey]", State_FileMap, FileMap_SchedulerInfo)
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return nil, errors.Wrapf(err, "[%v.%v:GetStorageLatest]", State_FileMap, FileMap_SchedulerInfo)
	}
	if !ok {
		return data, errors.Errorf("[%v.%v:GetStorageLatest value is nil]", State_FileMap, FileMap_SchedulerInfo)
	}
	return data, nil
}

func GetChallengesById(id uint64) ([]ChallengesInfo, int, error) {
	var (
		err  error
		data []ChallengesInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}
	b, err := types.EncodeToBytes(id)
	if err != nil {
		return nil, configs.Code_500, errors.Wrapf(err, "[EncodeToBytes]")
	}
	key, err := types.CreateStorageKey(meta, State_SegmentBook, SegmentBook_ChallengeMap, b)
	if err != nil {
		return nil, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return nil, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("value is empty")
	}
	return data, configs.Code_200, nil
}

//
func GetSchedulerPukFromChain() (Chain_SchedulerPuk, int, error) {
	var (
		err  error
		data Chain_SchedulerPuk
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_FileMap, FileMap_SchedulerPuk)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("value is empty")
	}
	return data, configs.Code_200, nil
}

func GetInvalidFileById(id uint64) ([]types.Bytes, int, error) {
	var (
		err  error
		data []types.Bytes
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_FileBank, FileBank_InvalidFile)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("value is empty")
	}
	return data, configs.Code_200, nil
}

// Query Scheduler info
func GetSchedulerInfoOnChain() ([]SchedulerInfo, int, error) {
	var (
		err   error
		mdata []SchedulerInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v.%v] [err:%v]", State_FileMap, FileMap_SchedulerInfo, err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_FileMap, FileMap_SchedulerInfo)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return mdata, configs.Code_404, errors.New("value is empty")
	}
	return mdata, configs.Code_200, nil
}

func GetFillerInfo(id types.U64, fileid string) (SpaceFileInfo, int, error) {
	var (
		err  error
		data SpaceFileInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v.%v] [err:%v]", State_FileBank, FileBank_FillerMap, err)
		}
	}()

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	b, err := types.EncodeToBytes(id)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[EncodeToBytes]")
	}
	ids, err := types.EncodeToBytes(fileid)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[EncodeToBytes]")
	}
	key, err := types.CreateStorageKey(meta, State_FileBank, FileBank_FillerMap, b, ids)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("value is empty")
	}
	return data, configs.Code_200, nil
}

// Get miner information on the cess chain
func ChainSt_Test(rpcaddr, signaturePrk, pallert, method string) error {
	var (
		err   error
		mdata SpaceFileInfo
	)
	api, err := gsrpc.NewSubstrateAPI(rpcaddr)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		return err
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, pallert, method)
	if err != nil {
		return errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return errors.New("empty")
	}
	fmt.Println(mdata)
	return nil
}
