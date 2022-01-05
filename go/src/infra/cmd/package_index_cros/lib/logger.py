import logging


def SetupLogger(level: int = logging.WARNING):
  """To be called once from main."""

  logger = logging.getLogger()

  logger.setLevel(level)

  formatter = logging.Formatter(
      '[%(levelname)s][%(filename)s:%(lineno)d] %(message)s')

  ch = logging.StreamHandler()
  ch.setLevel(level)
  ch.setFormatter(formatter)
  logger.addHandler(ch)

  return logger


g_logger = logging.getLogger()
