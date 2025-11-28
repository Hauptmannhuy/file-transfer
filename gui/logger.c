#include <pthread.h>
#include <stdlib.h>

#include "logger.h"

static pthread_mutex_t mutex = PTHREAD_MUTEX_INITIALIZER;

void u_logger_impl(char *file, int line, LOG_TYPE type, char *fmt, ...) {
  int mutex_result = 0;
  mutex_result = pthread_mutex_lock(&mutex);
  if (mutex_result == -1) {
    u_logger_error("error locking mutex");
  }
  char *str_type = "UNKNOWN";
  switch (type) {
  case log_info:
    str_type = "INFO";
    break;
  case log_warn:
    str_type = "WARN";
    break;
  case log_success:
    str_type = "SUCCESS";
    break;
  case log_error:
    str_type = "ERROR";
    break;
  default:
    break;
  }

  FILE *stream = type == log_error ? stderr : stdout;

  va_list args;
  va_start(args, fmt);
  fprintf(stream, "[%s:%d] %s: ", file, line, str_type);
  vfprintf(stream, fmt, args);
  va_end(args);
  fputc('\n', stream);

  mutex_result = pthread_mutex_unlock(&mutex);
  if (mutex_result == -1) {
    u_logger_error("error unlocking mutex");
  }
};
