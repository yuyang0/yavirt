package utils

import (
	"context"
	"fmt"
	"strconv"

	"github.com/projecteru2/yavirt/pkg/sh"
)

// CreateImage .
func CreateImage(ctx context.Context, fmt, path string, cap int64) error {
	return sh.ExecContext(ctx, "qemu-img", "create", "-q", "-f", fmt, path, strconv.FormatInt(cap, 10))
}

// AmplifyImage .
func AmplifyImage(ctx context.Context, path string, delta int64) error {
	flag := fmt.Sprintf("%+d", delta)
	return sh.ExecContext(ctx, "qemu-img", "resize", path, flag)
}

// CommitImage .
func CommitImage(ctx context.Context, path string) error {
	return sh.ExecContext(ctx, "qemu-img", "commit", path)
}

// CreateSnapshot .
func CreateSnapshot(ctx context.Context, volPath string, newVolPath string) error {
	return sh.ExecContext(ctx, "qemu-img", "create", "-f", "qcow2", "-F", "qcow2", newVolPath, "-b", volPath)
}

// RebaseImage .
func RebaseImage(ctx context.Context, volPath string, backingVolPath string) error {
	return sh.ExecContext(ctx, "qemu-img", "rebase", "-b", backingVolPath, volPath)
}

// Check .
func Check(ctx context.Context, volPath string) error {
	return sh.ExecContext(ctx, "qemu-img", "check", volPath)
}

// Repair .
func Repair(ctx context.Context, volPath string) error {
	return sh.ExecContext(ctx, "qemu-img", "check", "-r", "all", volPath)
}

// Write an image to a block device
func WriteImageToBlockDevice(ctx context.Context, imgPath string, device string) error {
	// two methods
	// qemu-img convert -f qcow2 -O raw my-qcow2.img /dev/sdb
	// qemu-img dd -f qcow2 -O raw bs=4M if=/vm-images/image.qcow2 of=/dev/sdd1
	//
	// for Ceph RBD, use following command:
	//    qemu-img convert -f qcow2 -O raw debian_squeeze.qcow2 rbd:data/squeeze
	return sh.ExecContext(ctx, "qemu-img", "convert", "-f", "qcow2", "-O", "raw", imgPath, device)
}
