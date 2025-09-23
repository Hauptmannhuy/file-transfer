#include "tpool.h"
#include "logger.h"
#include <stdlib.h>

#define reserved_thread 1

static thread_pool_work_t *create_work(thread_func_t func, void *arg) {
  if (func == NULL) {
    return NULL;
  }

  thread_pool_work_t *new_work = malloc(sizeof(thread_pool_work_t));

  new_work->func = func;
  new_work->arg = arg;
  new_work->next = NULL;

  return new_work;
};

static void destroy_work(thread_pool_work_t *work) {
  if (work == NULL) {
    return;
  }

  free(work);
}

static thread_pool_work_t *get_work(thread_pool_t *tpool) {
  thread_pool_work_t *work;

  if (tpool == NULL) {
    u_logger_error("thread pool is not initialized");
    abort();
  }

  work = tpool->work_first;

  if (work == NULL) {
    u_logger_info("work first is null");
    return NULL;
  }

  if (work->next == NULL) {
    tpool->work_first = NULL;
    tpool->work_last = NULL;
  } else {
    tpool->work_first = work->next;
  }

  return work;
}

static void *thread_pool_worker(void *arg) {
  thread_pool_t *tpool = arg;
  thread_pool_work_t *work;
  u_logger_info("thread created");
  while (1) {
    pthread_mutex_lock(&tpool->mutex);
    while (tpool->work_first == NULL && !tpool->stop) {
      u_logger_info("sleeping");
      pthread_cond_wait(&tpool->work_cond, &tpool->mutex);
    }
    u_logger_info("wake up...");
    if (tpool->stop) {
      break;
    }

    work = get_work(tpool);
    tpool->working_cnt++;

    pthread_mutex_unlock(&tpool->mutex);

    if (work != NULL) {
      work->func(work->arg);
      destroy_work(work);
    }

    pthread_mutex_lock(&tpool->mutex);
    tpool->working_cnt--;

    if (tpool->working_cnt == 0 && tpool->work_first == NULL && !tpool->stop) {
      pthread_cond_signal(&tpool->working_cond);
    }

    pthread_mutex_unlock(&tpool->mutex);
  }

  tpool->working_cnt--;
  pthread_cond_signal(&tpool->working_cond);
  pthread_mutex_unlock(&tpool->mutex);
  return NULL;
}

void tpool_wait(thread_pool_t *tpool) {
  if (tpool == NULL)
    return;

  pthread_mutex_lock(&(tpool->mutex));
  while (1) {
    if (tpool->work_first != NULL ||
        (!tpool->stop && tpool->working_cnt != 0) ||
        (tpool->stop && tpool->thread_cnt != 0)) {
      pthread_cond_wait(&(tpool->working_cond), &(tpool->mutex));
    } else {
      break;
    }
  }
  pthread_mutex_unlock(&(tpool->mutex));
}

bool tpool_add_work(thread_pool_t *tpool, thread_func_t func, void *arg) {
  if (tpool == NULL) {
    u_logger_warn("thread pool is not initialized");
    return false;
  }

  pthread_mutex_lock(&tpool->mutex);

  thread_pool_work_t *new_work = create_work(func, arg);

  if (tpool->work_first == NULL) {
    tpool->work_first = new_work;
    tpool->work_last = new_work;
  } else {
    tpool->work_last->next = new_work;
    tpool->work_last = new_work;
  }

  pthread_cond_broadcast(&tpool->work_cond);
  pthread_mutex_unlock(&tpool->mutex);

  return true;
}

void destroy_tpool(thread_pool_t *tpool) {
  if (tpool == NULL) {
    return;
  }

  thread_pool_work_t *work;
  thread_pool_work_t *work1;

  work = tpool->work_first;

  while (work != NULL) {
    work1 = work->next;
    destroy_work(work);
    work = work1;
  }

  tpool->stop = true;
  tpool->work_first = NULL;

  pthread_cond_broadcast(&(tpool->work_cond));
  pthread_mutex_unlock(&(tpool->mutex));

  tpool_wait(tpool);

  pthread_mutex_destroy(&(tpool->mutex));
  pthread_cond_destroy(&(tpool->work_cond));
  pthread_cond_destroy(&(tpool->working_cond));

  free(tpool);
}

thread_pool_t *create_tpool(int thread_num) {
  thread_pool_t *tpool = malloc(sizeof(thread_pool_t));
  pthread_t thread;

  tpool->work_first = NULL;
  tpool->work_last = NULL;
  tpool->stop = false;

  tpool->thread_cnt = thread_num;

  pthread_mutex_init(&(tpool->mutex), NULL);
  pthread_cond_init(&(tpool->work_cond), NULL);
  pthread_cond_init(&(tpool->working_cond), NULL);

  for (int i = 0; i < reserved_thread + thread_num; i++) {
    pthread_create(&thread, NULL, thread_pool_worker, (void *)(tpool));
  }
  u_logger_info("thread pool created");
  return tpool;
}
