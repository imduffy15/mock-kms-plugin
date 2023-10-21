package services

import (
	"context"
	"math/rand"
	"net"
	"time"

	"google.golang.org/grpc"

	"k8s.io/klog/v2"
	"k8s.io/kms/apis/v1beta1"
)

var _ v1beta1.KeyManagementServiceServer = &NopService{}

type NopService struct {
	addr                 string
	timeout              time.Duration
	server               *grpc.Server
	encryptionLatencyMin time.Duration
	decryptionLatencyMin time.Duration
	encryptionLatencyMax time.Duration
	decryptionLatencyMax time.Duration
}

func generateRandomDuration(min, max time.Duration) time.Duration {
	if min > max {
		panic("Minimum duration cannot be greater than maximum duration")
	}

	rand.Seed(time.Now().UnixNano())
	durationRange := int64(max - min)
	randomNanos := int64(0)
	if durationRange != 0 {
		randomNanos = rand.Int63n(durationRange)
	}
	randomDuration := min + time.Duration(randomNanos)
	return randomDuration
}

func (n *NopService) Version(ctx context.Context, request *v1beta1.VersionRequest) (*v1beta1.VersionResponse, error) {
	klog.Info("Received request for version: %v", request)
	return &v1beta1.VersionResponse{
		Version:        "v1beta1",
		RuntimeName:    "mock-kms-plugin",
		RuntimeVersion: "0.0.1",
	}, nil
}

func (n *NopService) Decrypt(ctx context.Context, request *v1beta1.DecryptRequest) (*v1beta1.DecryptResponse, error) {
	randomDelay := generateRandomDuration(n.decryptionLatencyMin, n.decryptionLatencyMax)

	klog.InfoS(
		"Received Decrypt Request",
		"cipher", string(request.Cipher),
		"latency", randomDelay,
	)

	if string(request.Cipher) != "ping" {
		time.Sleep(randomDelay)
	}

	return &v1beta1.DecryptResponse{Plain: request.Cipher}, nil
}

func (n *NopService) Encrypt(ctx context.Context, request *v1beta1.EncryptRequest) (*v1beta1.EncryptResponse, error) {
	randomDelay := generateRandomDuration(n.encryptionLatencyMin, n.encryptionLatencyMax)

	klog.InfoS(
		"Received Encrypt Request",
		"request", request.Plain,
		"latency", randomDelay,
	)

	if string(request.Plain) != "ping" {
		time.Sleep(randomDelay)
	}

	return &v1beta1.EncryptResponse{Cipher: request.Plain}, nil
}
func NewNopService(
	address string,
	timeout time.Duration,
	decryptionLatencyMin time.Duration,
	decryptionLatencyMax time.Duration,
	encryptionLatencyMin time.Duration,
	encryptionLatencyMax time.Duration,
) *NopService {
	klog.InfoS("KMS plugin configured", "address", address, "timeout", timeout)

	return &NopService{
		addr:                 address,
		timeout:              timeout,
		encryptionLatencyMin: encryptionLatencyMin,
		decryptionLatencyMin: decryptionLatencyMin,
		encryptionLatencyMax: encryptionLatencyMax,
		decryptionLatencyMax: decryptionLatencyMax,
	}
}

// ListenAndServe accepts incoming connections on a Unix socket. It is a blocking method.
// Returns non-nil error unless Close or Shutdown is called.
func (n *NopService) ListenAndServe() error {
	ln, err := net.Listen("unix", n.addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	gs := grpc.NewServer(
		grpc.ConnectionTimeout(n.timeout),
	)
	n.server = gs

	v1beta1.RegisterKeyManagementServiceServer(gs, n)

	klog.InfoS("kms plugin serving", "address", n.addr)
	return gs.Serve(ln)
}

func (n *NopService) Shutdown() {
	klog.V(4).InfoS("kms plugin shutdown", "address", n.addr)
	if n.server != nil {
		n.server.GracefulStop()
	}
}
func (n *NopService) Close() {
	klog.V(4).InfoS("kms plugin close", "address", n.addr)
	if n.server != nil {
		n.server.Stop()
	}
}
