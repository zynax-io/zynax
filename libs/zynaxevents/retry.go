// SPDX-License-Identifier: Apache-2.0

package zynaxevents

import "time"

// RetryBackoff is the ordered list of retry delays applied to NATS JetStream
// consumer redelivery attempts. Five entries align with MaxDeliver=5.
// After the fifth delivery attempt the message is forwarded to the DLQ subject.
// Pinned by testdata/golden/retry_policy.json.
var RetryBackoff = []time.Duration{
	1 * time.Second,
	5 * time.Second,
	30 * time.Second,
	2 * time.Minute,
	5 * time.Minute,
}
