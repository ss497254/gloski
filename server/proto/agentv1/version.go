package agentv1

// ProtocolVersion is raised whenever master and agent can no longer talk to
// each other across the change. A master refuses agents with another value;
// the refused agent exits and the master installs its own build instead.
const ProtocolVersion = 4
