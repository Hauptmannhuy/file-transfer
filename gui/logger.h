#include <stdarg.h>
#include <stdio.h>
#include <string.h>
#pragma once

typedef enum { log_error, log_success, log_warn, log_info } LOG_TYPE;
void u_logger_impl(char *file, int line, LOG_TYPE type, char *fmt, ...);

#define u_logger(type, fmt, ...)                                               \
  u_logger_impl(__FILE__, __LINE__, type, fmt, ##__VA_ARGS__)

#define u_logger_info(fmt, ...)                                                \
  u_logger_impl(__FILE__, __LINE__, log_info, fmt, ##__VA_ARGS__)

#define u_logger_warn(fmt, ...)                                                \
  u_logger_impl(__FILE__, __LINE__, log_warn, fmt, ##__VA_ARGS__)

#define u_logger_success(fmt, ...)                                             \
  u_logger_impl(__FILE__, __LINE__, log_success, fmt, ##__VA_ARGS__)

#define u_logger_error(fmt, ...)                                               \
  u_logger_impl(__FILE__, __LINE__, log_error, fmt, ##__VA_ARGS__)
