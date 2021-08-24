# pylint: skip-file
"""
This file holds simple dict utilities globally constant across files.
"""
import logging
import operator

OPERATIONS_MAP = {
  '<': operator.lt,
  '<=': operator.le,
  '==': operator.eq,
  '!=': operator.ne,
  '>=': operator.ge,
  '>': operator.gt
}

LOGGING_LEVELS = {
    0: logging.ERROR,
    1: logging.WARNING,
    2: logging.INFO,
    3: logging.DEBUG
}

