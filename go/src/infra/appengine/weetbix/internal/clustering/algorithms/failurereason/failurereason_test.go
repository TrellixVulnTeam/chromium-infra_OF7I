// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package failurereason

import (
	"testing"

	"infra/appengine/weetbix/internal/bugclusters/rules"
	"infra/appengine/weetbix/internal/clustering"
	pb "infra/appengine/weetbix/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAlgorithm(t *testing.T) {
	Convey(`Cluster`, t, func() {
		a := &Algorithm{}
		Convey(`Does not cluster test result without failure reason`, func() {
			id := a.Cluster(&clustering.Failure{})
			So(id, ShouldBeNil)
		})
		Convey(`ID of appropriate length`, func() {
			id := a.Cluster(&clustering.Failure{
				Reason: &pb.FailureReason{PrimaryErrorMessage: "abcd this is a test failure message"},
			})
			// IDs may be 16 bytes at most.
			So(len(id), ShouldBeGreaterThan, 0)
			So(len(id), ShouldBeLessThanOrEqualTo, clustering.MaxClusterIDBytes)
		})
		Convey(`Same ID for same cluster with different numbers`, func() {
			id1 := a.Cluster(&clustering.Failure{
				Reason: &pb.FailureReason{PrimaryErrorMessage: "Null pointer exception at ip 0x45637271"},
			})
			id2 := a.Cluster(&clustering.Failure{
				Reason: &pb.FailureReason{PrimaryErrorMessage: "Null pointer exception at ip 0x12345678"},
			})
			So(id2, ShouldResemble, id1)
		})
		Convey(`Different ID for different clusters`, func() {
			id1 := a.Cluster(&clustering.Failure{
				Reason: &pb.FailureReason{PrimaryErrorMessage: "Exception in TestMethod"},
			})
			id2 := a.Cluster(&clustering.Failure{
				Reason: &pb.FailureReason{PrimaryErrorMessage: "Exception in MethodUnderTest"},
			})
			So(id2, ShouldNotResemble, id1)
		})
	})
	Convey(`Failure Association Rule`, t, func() {
		a := &Algorithm{}
		test := func(failure *clustering.Failure, expectedRule string) {
			rule := a.FailureAssociationRule(failure)
			So(rule, ShouldEqual, expectedRule)

			// Test the rule is valid syntax and matches at least the example failure.
			expr, err := rules.Parse(rule, "reason")
			So(err, ShouldBeNil)
			So(expr.Evaluate(map[string]string{
				"reason": failure.Reason.PrimaryErrorMessage,
			}), ShouldBeTrue)
		}
		Convey(`Hexadecimal`, func() {
			failure := &clustering.Failure{
				Reason: &pb.FailureReason{PrimaryErrorMessage: "Null pointer exception at ip 0x45637271"},
			}
			test(failure, `reason LIKE "Null pointer exception at ip %"`)
		})
		Convey(`Numeric`, func() {
			failure := &clustering.Failure{
				Reason: &pb.FailureReason{PrimaryErrorMessage: "Could not connect to 127.1.2.1: connection refused"},
			}
			test(failure, `reason LIKE "Could not connect to %.%.%.%: connection refused"`)
		})
		Convey(`Base64`, func() {
			failure := &clustering.Failure{
				Reason: &pb.FailureReason{PrimaryErrorMessage: "Received unexpected response: AdafdxAAD17917+/="},
			}
			test(failure, `reason LIKE "Received unexpected response: %"`)
		})
		Convey(`Escaping`, func() {
			failure := &clustering.Failure{
				Reason: &pb.FailureReason{PrimaryErrorMessage: `_%"'+[]|` + "\u0000\r\n\v\u202E\u2066 AdafdxAAD17917+/="},
			}
			test(failure, `reason LIKE "\\_\\%\"'+[]|\x00\r\n\v\u202e\u2066 %"`)
		})
	})
	Convey(`Cluster Description`, t, func() {
		a := &Algorithm{}
		Convey(`Hexadecimal`, func() {
			failure := &clustering.Failure{
				Reason: &pb.FailureReason{PrimaryErrorMessage: "Null pointer exception at ip 0x45637271"},
			}
			description := a.ClusterDescription(failure)
			So(description.Title, ShouldEqual, `Null pointer exception at ip 0x45637271`)
			So(description.Description, ShouldContainSubstring, `Null pointer exception at ip 0x45637271`)
		})
		Convey(`Escaping`, func() {
			failure := &clustering.Failure{
				Reason: &pb.FailureReason{PrimaryErrorMessage: `_%"'+[]|` + "\u0000\r\n\v\u202E\u2066 AdafdxAAD17917+/="},
			}
			description := a.ClusterDescription(failure)
			So(description.Title, ShouldEqual, `_%\"'+[]|\x00\r\n\v\u202e\u2066 AdafdxAAD17917+/=`)
			So(description.Description, ShouldContainSubstring, `_%\"'+[]|\x00\r\n\v\u202e\u2066 AdafdxAAD17917+/=`)
		})
	})
}
