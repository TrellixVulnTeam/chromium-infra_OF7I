# pylint: skip-file
""" This file exists to filter against the target file run for python_finder:
effectively, all items here exist to filter out libraries which are common to all
basic python programs from the target file so as to not have to find wheel
requirements for them.
"""

import os
import sys


def main() -> None:
  pass


if __name__ == "__main__":
  main()
