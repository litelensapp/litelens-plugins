package config

// PluginID identifies this plugin to the host app — used as the sender ID on
// emitted events and to build the ack topic the host's PublishAndAwaitAck
// waits on.
const PluginID = "helm"
