#define _GNU_SOURCE
#include <fcntl.h>
#include <sched.h>
#include <unistd.h>
#include <stdlib.h>
#include <stdio.h>
#include <sys/mount.h>
#include <string.h>
#include <errno.h>
#include <time.h>
#include <sys/timeb.h>

int usage()
{
	printf("mount_cgroup pid Src Dst\n");
	return 0;
}

int main(int argc, char* argv[])
{
	int ret = 0;
	char mnt_path[256];
	char *pid = NULL;
	char *src = NULL, *dst = NULL;

	if (argc < 3 || !strcmp(argv[1], "-h")) {
		return usage();
	}

	pid = argv[1];
	src = argv[2];
	dst = argv[3];

	sprintf(mnt_path, "/proc/%s/ns/mnt", pid);
	int fd = open(mnt_path, O_RDONLY | O_CLOEXEC);
	if (fd < 0) {
		printf("Can not open mnt path:%s\n", mnt_path);
		return -ENOMEM;
	}

	printf("entered namespace :%s\n", mnt_path);

	if (setns(fd, 0) < 0) {
		printf("set ns failed\n");
		return -ENOMEM;
	}

	struct timeb timer_msec;
	long long int timestamp_msec; /* timestamp in millisecond. */
	if (!ftime(&timer_msec)) {
		timestamp_msec = ((long long int)timer_msec.time) * 1000ll + (long long int)timer_msec.millitm;
	} else {
		timestamp_msec = -1;
	}
	printf("%lld milliseconds since epoch\n", timestamp_msec);

//	 int gpu_holder = open(dst, O_CREAT | O_RDONLY, 0755);
//     printf("gpu_holder:%d，%d\n",gpu_holder,errno);
//     close(gpu_holder);
	if (ret = mount(src, dst, NULL, MS_BIND, NULL) < 0) {
		printf("mount from %s to %s failed:%d\n", src, dst, errno);
		return ret;
	}

	printf("Successfully bind mount for %s-->%s\n", src, dst);
	return 0;
}

