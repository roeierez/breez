package bindings

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/breez/breez"
	"github.com/breez/breez/bootstrap"
	"github.com/breez/breez/data"
	"github.com/breez/breez/doubleratchet"
	breezlog "github.com/breez/breez/log"
	"github.com/breez/breez/sync"
	"github.com/btcsuite/btclog"
	"github.com/golang/protobuf/proto"
)

var (
	appServices AppServices
)

// AppServices defined the interface needed in Breez library in order to functional
// right.
type AppServices interface {
	Notify(notificationEvent []byte)
	BackupProviderName() string
	BackupProviderSignIn() (string, error)
}

// Logger is an interface that is used to log to the central log file.
type Logger interface {
	Log(msg string, lvl string)
}

// BreezLogger is the implementation of Logger
type BreezLogger struct {
	log btclog.Logger
}

// BackupService is an service to this libary for backup execution.
type BackupService interface {
	Backup(files string, nodeID, backupID string) error
}

// Log writs to the centeral log file
func (l *BreezLogger) Log(msg string, lvl string) {
	switch lvl {
	case "FINEST":
	case "FINER":
	case "FINE":
		l.log.Tracef(msg)
	case "CONFIG":
		l.log.Debugf(msg)
	case "INFO":
		l.log.Infof(msg)
	case "WARNING":
		l.log.Warnf(msg)
	case "SEVERE":
		l.log.Errorf(msg)
	case "SHOUT":
		l.log.Criticalf(msg)
	default:
		l.log.Infof(msg)
	}
}

/*
JobController is the interface to return when scheuling a job to allow the caller to cancel at
any time
*/
type JobController interface {
	Run() error
	Stop()
}

/*
Init initialize lightning client
*/
func Init(tempDir string, workingDir string, services AppServices) error {
	os.Setenv("TMPDIR", tempDir)
	appServices = services
	return breez.Init(workingDir, services)
}

/*
Start the lightning client
*/
func Start() (err error) {
	notificationsChan, err := breez.Start()
	if err != nil {
		return err
	}
	go deliverNotifications(notificationsChan, appServices)
	return nil
}

/*
NewSyncJob starts breez only to reach synchronized state.
The daemon closes itself automatically when reaching this state.
*/
func NewSyncJob(workingDir string) (JobController, error) {
	job, err := sync.NewJob(workingDir)
	if err != nil {
		return nil, err
	}
	return job, nil
}

/*
GetLogger creates a logger that logs to the same breez central log file
*/
func GetLogger(appDir string) (Logger, error) {
	backend, err := breezlog.GetLogBackend(appDir)
	if err != nil {
		return nil, err
	}
	logger := backend.Logger("BIND")
	return &BreezLogger{logger}, nil
}

/*
Stop the lightning client
*/
func Stop() {
	breez.Stop()
}

/*
RequestBackup triggers breez RequestBackup
*/
func RequestBackup() {
	breez.RequestBackup()
}

/*
RestoreBackup is part of the binding inteface which is delegated to breez.RestoreBackup
*/
func RestoreBackup(nodeID string) error {
	return breez.Restore(nodeID)
}

/*
AvailableSnapshots is part of the binding inteface which is delegated to breez.AvailableSnapshots
*/
func AvailableSnapshots(nodeID string) (string, error) {
	snapshots, err := breez.AvailableSnapshots()
	if err != nil {
		return "", err
	}
	bytes, err := json.Marshal(snapshots)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

/*
WaitDaemonShutdown blocks untill the daemon shutdown
*/
func WaitDaemonShutdown() {
	breez.WaitDaemonShutdown()
}

/*
DaemonReady returns the status of the daemon
*/
func DaemonReady() bool {
	return breez.DaemonReady()
}

/*
OnResume just calls the breez.OnResume
*/
func OnResume() {
	breez.OnResume()
}

/*
Log is a function that uses the breez logger
*/
func Log(msg string, lvl string) {
	breez.Log(msg, lvl)
}

/*
GetAccountInfo is part of the binding inteface which is delegated to breez.GetAccountInfo
*/
func GetAccountInfo() ([]byte, error) {
	return marshalResponse(breez.GetAccountInfo())
}

/*
ConnectAccount is part of the binding inteface which is delegated to breez.ConnectAccount
*/
func ConnectAccount() error {
	return breez.ConnectAccount()
}

/*
IsConnectedToRoutingNode is part of the binding inteface which is delegated to breez.IsConnectedToRoutingNode
*/
func IsConnectedToRoutingNode() bool {
	return breez.IsConnectedToRoutingNode()
}

/*
AddFundsInit is part of the binding inteface which is delegated to breez.AddFundsInit
*/
func AddFundsInit(breezID string) ([]byte, error) {
	return marshalResponse(breez.AddFundsInit(breezID))
}

/*
GetRefundableSwapAddresses returns all addresses that are refundable, e.g expired and not paid
*/
func GetRefundableSwapAddresses() ([]byte, error) {
	fmt.Println("GetRefundableSwapAddresses in api")
	refundableAddresses, err := breez.GetRefundableAddresses()
	if err != nil {
		fmt.Println("GetRefundableSwapAddresses in api returned error from breez")
		return nil, err
	}

	var rpcAddresses []*data.SwapAddressInfo
	for _, a := range refundableAddresses {
		rpcAddresses = append(rpcAddresses, &data.SwapAddressInfo{
			Address:                 a.Address,
			PaymentHash:             hex.EncodeToString(a.PaymentHash),
			ConfirmedAmount:         a.ConfirmedAmount,
			ConfirmedTransactionIds: a.ConfirmedTransactionIds,
			PaidAmount:              a.PaidAmount,
			LockHeight:              a.LockHeight,
			ErrorMessage:            a.ErrorMessage,
			LastRefundTxID:          a.LastRefundTxID,
		})
	}

	addressList := &data.SwapAddressList{
		Addresses: rpcAddresses,
	}
	fmt.Printf("GetRefundableSwapAddresses returned %v addresses", len(rpcAddresses))
	return marshalResponse(addressList, nil)
}

//Refund transfers the funds in address to the user destination address
func Refund(refundRequest []byte) (string, error) {
	request := &data.RefundRequest{}
	if err := proto.Unmarshal(refundRequest, request); err != nil {
		return "", err
	}
	return breez.Refund(request.Address, request.RefundAddress)
}

/*
GetFundStatus is part of the binding inteface which is delegated to breez.GetFundStatus
*/
func GetFundStatus(notificationToken string) ([]byte, error) {
	return marshalResponse(breez.GetFundStatus(notificationToken))
}

/*
RemoveFund is part of the binding inteface which is delegated to breez.RemoveFund
*/
func RemoveFund(removeFundRequest []byte) ([]byte, error) {
	request := &data.RemoveFundRequest{}
	proto.Unmarshal(removeFundRequest, request)
	return marshalResponse(breez.RemoveFund(request.Amount, request.Address))
}

/*
GetLogPath is part of the binding inteface which is delegated to breez.GetLogPath
*/
func GetLogPath() string {
	return breez.GetLogPath()
}

/*
GetPayments is part of the binding inteface which is delegated to breez.GetPayments
*/
func GetPayments() ([]byte, error) {
	return marshalResponse(breez.GetPayments())
}

/*
PayBlankInvoice is part of the binding inteface which is delegated to breez.PayBlankInvoice
*/
func SendPaymentForRequest(payInvoiceRequest []byte) error {
	decodedRequest := &data.PayInvoiceRequest{}
	proto.Unmarshal(payInvoiceRequest, decodedRequest)
	return breez.SendPaymentForRequest(decodedRequest.PaymentRequest, decodedRequest.Amount)
}

/*
AddInvoice is part of the binding inteface which is delegated to breez.AddInvoice
*/
func AddInvoice(invoice []byte) (paymentRequest string, err error) {
	decodedInvoiceMemo := &data.InvoiceMemo{}
	proto.Unmarshal(invoice, decodedInvoiceMemo)
	return breez.AddInvoice(decodedInvoiceMemo)
}

/*
AddStandardInvoice is part of the binding inteface which is delegated to breez.AddStandardInvoice
*/
func AddStandardInvoice(invoice []byte) (paymentRequest string, err error) {
	decodedStandardInvoiceMemo := &data.InvoiceMemo{}
	proto.Unmarshal(invoice, decodedStandardInvoiceMemo)
	return breez.AddStandardInvoice(decodedStandardInvoiceMemo)
}

/*
DecodePaymentRequest is part of the binding inteface which is delegated to breez.DecodePaymentRequest
*/
func DecodePaymentRequest(paymentRequest string) ([]byte, error) {
	return marshalResponse(breez.DecodePaymentRequest(paymentRequest))
}

/*
GetRelatedInvoice is part of the binding inteface which is delegated to breez.GetRelatedInvoice
*/
func GetRelatedInvoice(paymentRequest string) ([]byte, error) {
	return marshalResponse(breez.GetRelatedInvoice(paymentRequest))
}

/*
SendWalletCoins is part of the binding inteface which is delegated to breez.SendWalletCoins
*/
func SendWalletCoins(sendCoinsRequest []byte) (string, error) {
	unmarshaledRequest := data.SendWalletCoinsRequest{}
	proto.Unmarshal(sendCoinsRequest, &unmarshaledRequest)
	return breez.SendWalletCoins(unmarshaledRequest.Address, unmarshaledRequest.Amount, unmarshaledRequest.SatPerByteFee)
}

/*
GetDefaultOnChainFeeRate is part of the binding inteface which is delegated to breez.GetDefaultOnChainFeeRate
*/
func GetDefaultOnChainFeeRate() int64 {
	return breez.GetDefaultSatPerByteFee()
}

/*
ValidateAddress is part of the binding inteface which is delegated to breez.ValidateAddress
*/
func ValidateAddress(address string) error {
	return breez.ValidateAddress(address)
}

/*
SendCommand is part of the binding inteface which is delegated to breez.SendPaymentForRequest
*/
func SendCommand(command string) (string, error) {
	return breez.SendCommand(command)
}

/*
RegisterReceivePaymentReadyNotification is part of the binding inteface which is delegated to breez.RegisterReceivePaymentReadyNotification
*/
func RegisterReceivePaymentReadyNotification(token string) error {
	return breez.RegisterReceivePaymentReadyNotification(token)
}

/*
RegisterChannelOpenedNotification is part of the binding inteface which is delegated to breez.RegisterChannelOpenedNotification
*/
func RegisterChannelOpenedNotification(token string) error {
	return breez.RegisterChannelOpenedNotification(token)
}

/*
RegisterPeriodicSync is part of the binding inteface which is delegated to breez.RegisterPeriodicSync
*/
func RegisterPeriodicSync(token string) error {
	return breez.RegisterPeriodicSync(token)
}

/*
CreateRatchetSession is part of the binding inteface which is delegated to breez.CreateRatchetSession
*/
func CreateRatchetSession(request []byte) ([]byte, error) {
	var err error
	var secret, pubKey string

	unmarshaledRequest := &data.CreateRatchetSessionRequest{}
	if err := proto.Unmarshal(request, unmarshaledRequest); err != nil {
		return nil, err
	}

	//if has secret then we are initiators
	if unmarshaledRequest.Secret == "" {
		secret, pubKey, err = doubleratchet.NewSession(unmarshaledRequest.SessionID, unmarshaledRequest.Expiry)
	} else {
		err = doubleratchet.NewSessionWithRemoteKey(unmarshaledRequest.SessionID, unmarshaledRequest.Secret, unmarshaledRequest.RemotePubKey, unmarshaledRequest.Expiry)
	}

	if err != nil {
		return nil, err
	}
	return marshalResponse(&data.CreateRatchetSessionReply{SessionID: unmarshaledRequest.SessionID, Secret: secret, PubKey: pubKey}, nil)
}

/*
RatchetSessionInfo is part of the binding inteface which is delegated to breez.RatchetSessionInfo
*/
func RatchetSessionInfo(sessionID string) ([]byte, error) {
	var reply *data.RatchetSessionInfoReply
	sessionDetails := doubleratchet.RatchetSessionInfo(sessionID)
	if sessionDetails == nil {
		reply = &data.RatchetSessionInfoReply{
			SessionID: "",
			Initiated: false,
		}
	} else {
		reply = &data.RatchetSessionInfoReply{
			SessionID: sessionDetails.SessionID,
			Initiated: sessionDetails.Initiated,
			UserInfo:  sessionDetails.UserInfo,
		}
	}
	return marshalResponse(reply, nil)
}

/*
RatchetSessionSetInfo is part of the binding inteface which is delegated to breez.RatchetSessionSetInfo
*/
func RatchetSessionSetInfo(request []byte) error {
	unmarshaledRequest := &data.RatchetSessionSetInfoRequest{}
	if err := proto.Unmarshal(request, unmarshaledRequest); err != nil {
		return err
	}
	return doubleratchet.RatchetSessionSetInfo(unmarshaledRequest.SessionID, unmarshaledRequest.UserInfo)
}

/*
RatchetEncrypt is part of the binding inteface which is delegated to breez.RatchetEncrypt
*/
func RatchetEncrypt(request []byte) (string, error) {
	unmarshaledRequest := &data.RatchetEncryptRequest{}
	if err := proto.Unmarshal(request, unmarshaledRequest); err != nil {
		return "", err
	}

	return doubleratchet.RatchetEncrypt(unmarshaledRequest.SessionID, unmarshaledRequest.Message)
}

/*
RatchetDecrypt is part of the binding inteface which is delegated to breez.RatchetDecrypt
*/
func RatchetDecrypt(request []byte) (string, error) {
	unmarshaledRequest := &data.RatchetDecryptRequest{}
	if err := proto.Unmarshal(request, unmarshaledRequest); err != nil {
		return "", err
	}

	return doubleratchet.RatchetDecrypt(unmarshaledRequest.SessionID, unmarshaledRequest.EncryptedMessage)
}

// BootstrapFiles is part of the binding inteface which is delegated to bootstrap.PutFiles
func BootstrapFiles(request []byte) error {
	req := &data.BootstrapFilesRequest{}
	if err := proto.Unmarshal(request, req); err != nil {
		return err
	}

	return bootstrap.PutFiles(req.GetWorkingDir(), req.GetFullPaths())
}

func deliverNotifications(notificationsChan chan data.NotificationEvent, appServices AppServices) {
	for {
		notification := <-notificationsChan
		res, err := proto.Marshal(&notification)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error in marshaing notification", err)
		}
		appServices.Notify(res)
	}
}

func marshalResponse(message proto.Message, responseError error) (buffer []byte, err error) {
	if responseError != nil {
		return nil, responseError
	}
	res, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}
	return res, nil
}
