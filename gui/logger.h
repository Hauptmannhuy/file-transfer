#include <stdarg.h>
#include <stdio.h>
#include <string.h>
#pragma once

typedef enum { error, success, warn, info } LOG_TYPE;
void u_logger_impl(char *file, int line, LOG_TYPE type, char *fmt, ...);

#define u_logger(type, fmt, ...)                                               \
  u_logger_impl(__FILE__, __LINE__, type, fmt, ##__VA_ARGS__)

#define u_logger_info(fmt, ...)                                                \
  u_logger_impl(__FILE__, __LINE__, info, fmt, ##__VA_ARGS__)

#define u_logger_warn(fmt, ...)                                                \
  u_logger_impl(__FILE__, __LINE__, warn, fmt, ##__VA_ARGS__)

#define u_logger_success(fmt, ...)                                             \
  u_logger_impl(__FILE__, __LINE__, success, fmt, ##__VA_ARGS__)

#define u_logger_error(fmt, ...)                                               \
  u_logger_impl(__FILE__, __LINE__, error, fmt, ##__VA_ARGS__)
