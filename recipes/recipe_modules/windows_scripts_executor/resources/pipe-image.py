#! /usr/bin/python3

from typing import Sequence

import argparse
import json
import sys

description = ''' Pipe the given json into input properties of a led job
                  description. Can be used to chain in the led request. For
                  example:
                    led get-builder infra-internal/try:"WinPE Customization Builder" |
                    led edit-recipe-bundle |
                    pipe-image.py --p /<configs-path>/win10_regedit.json |
                    led launch
              '''
parser = argparse.ArgumentParser(description=description)
parser.add_argument(
    '--p',
    metavar='image.json',
    type=str,
    nargs='+',
    help='JSON file to insert in input properties')
args = parser.parse_args()

pipe_props = json.load(sys.stdin)
for prop in args.p:
  with open(prop, 'r') as f:
    input_props = json.load(f)
    for k, v in input_props.items():
      pipe_props['buildbucket']['bbagent_args']['build']['input']['properties'][
          k] = v

json.dump(pipe_props, sys.stdout, indent=2)
