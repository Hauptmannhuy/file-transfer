#include "logger.h"

void u_logger_impl(char *file, int line, LOG_TYPE type, char *fmt, ...) {
  char *str_type;
  switch (type) {
  case info:
    str_type = "INFO";
    break;
  case warn:
    str_type = "WARN";
    break;
  case success:
    str_type = "SUCCESS";
    break;
  case error:
    str_type = "ERROR";
    break;
  default:
    break;
  }

  va_list args;
  va_start(args, fmt);
  char
      new_fmt[strlen(file) + sizeof(line) + strlen(str_type) + strlen(fmt) + 2];
  sprintf(new_fmt, "[%s:%d] %s: %s\n", file, line, str_type, fmt);
  vfprintf(stdout, new_fmt, args);
  va_end(args);
};