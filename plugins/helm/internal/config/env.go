package config

import "os"

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// GetHostGRPCPort returns the LITELENS_HOST_GRPC_PORT environment variable, the
// port of the host app's gRPC server that this plugin subprocess subscribes to
// for cluster-context change notifications (see kube.WatchClusterContext).
// Empty when unset, meaning the host didn't wire up the watch (e.g. running the
// plugin binary standalone outside the host app).
func GetHostGRPCPort() string {
	return getEnvOrDefault("LITELENS_HOST_GRPC_PORT", "")
}
