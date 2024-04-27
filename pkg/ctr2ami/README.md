# ctr2ami

## Requirements

`mkfs.ext4`

`blkid`

`guestmount`

## Notes

Create partitions

```
dd if=/dev/zero of=heyo-1 bs=2097152 count=1024

# Creat 256MB EFI partition.
parted heyo-1 mkpart primary fat32 2048s 501760s
parted heyo-1 set 1 boot on

# Create remaining partition.
```

New way with guestfish:

```
#### sparse heyo-1 10G    # Create a new sparse 10G disk.

disk-create heyo.img raw 10G preallocation:sparse
add heyo.img
run                  # This "activates" the disk and creates /dev/sda.

# Create the GPT label.
part-init /dev/sda gpt

# Add partition 1.
part-add /dev/sda primary 2048 501760
part-set-bootable /dev/sda 1 true
mkfs vfat /dev/sda1 label:EFI

# Add partition 2.
part-add /dev/sda primary 501761 20971486
mkfs ext4 /dev/sda2 label:ROOT
```

## Qemu commands

```
qemu-system-x86_64 \
  -enable-kvm \
  -cpu host,kvm=off \
  -m 2048 \
  -device nvme,drive=nvme0,serial=deadbeaf1,max_ioqpairs=8 \
  -drive if=none,id=nvme0,format=raw,media=disk,file=/home/joseph/.ctr2ami/b8132df8c2fc73f4c1e7ce434c1ff19b134818e8173cd5e8f79c55a5f635d7e5/vm.img \
  -drive if=pflash,format=raw,unit=0,readonly=on,file=OVMF_CODE.fd \
  -device e1000,netdev=user.0 \
  -netdev user,id=user.0 \
  -nographic \
  -vga none
```
