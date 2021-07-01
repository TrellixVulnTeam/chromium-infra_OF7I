// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

// Note: This plan created only as example and will be replaced by correct one by further CL.

// Default config for recovery instances.
// Please change careful to avoid breakage.
const defaultPlans = `
{
  "repair_dut": {
    "verifiers": [
      "dut_ssh"
    ],
    "actions": {
      "dut_rpm_info": {
        "allow_fail": true
      },
      "dut_ping":{
        "recoveries": [
          "sample_pass",
          "sample_pass",
          "servod_dut_cold_reset"
        ],
        "cache_strategy":"never"
      },
      "dut_ssh": {
        "recoveries": [
          "sample_pass",
          "servod_dut_cold_reset",
          "sample_pass"
        ],
        "dependencies": [
          "dut_ping"
        ]
      },
      "servo_host_servod_stop": {
        "allow_fail": true
      }
    }
  },
  "repair_servo": {
    "verifiers": [
      "servo_host_ssh",
      "servod_lidopen"
    ],
    "actions": {
      "servo_host_ssh": {
        "dependencies": [
          "servo_host_ping"
        ]
      },
      "servod_lidopen": {
        "dependencies": [
          "servod_echo"
        ],
        "recoveries": [
          "servod_lidopen_recover",
          "servo_host_servod_restart",
          "sample_pass"
        ]
      },
      "servod_lidopen_recover":{
        "dependencies": [
          "servod_echo"
        ]
      },
      "servo_host_servod_init": {
        "dependencies":[
          "servo_host_ssh"
        ],
        "recoveries": [
          "servo_host_servod_restart"
        ]
      },
      "servod_echo": {
        "dependencies":[
          "servo_host_servod_init"
        ],
        "recoveries": [
          "servo_host_servod_restart"
        ]
      },
      "servo_host_servod_restart":{
        "cache_strategy":"never"
      }
    },
    "allow_fail": true
  },
  "repair_labstation": {
    "verifiers": [
      "dut_ssh"
    ],
    "actions": {
      "dut_rpm_info": {
        "allow_fail": true
      },
      "dut_ssh": {
        "dependencies": [
          "dut_ping",
          "sample_pass"
        ]
      }
    }
  }
}
`
