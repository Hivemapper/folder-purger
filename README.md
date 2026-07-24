# folder-purger

`folder-purger` is a small service for keeping one or more directories under
configured size limits.

It is used on-device by `hivemapper-folder-purger.service` to prevent recording
directories from growing without bound.

## Usage

Pass one or more `path max-size` pairs:

```sh
folder-purger /data/recording/unprocessed_framekm 8000000000 /data/video 6500000000
```

Each `max-size` is interpreted as bytes by default. A value ending in `%` is
interpreted as a percentage of the filesystem size at startup.

## Behavior

On startup, and then every five minutes, the purger:

1. Computes the total size of each configured directory.
2. Includes files in subdirectories when calculating size.
3. If the directory is over its configured limit, removes 80% of its top-level
   items, oldest `mtime` first.

Top-level items can be files or directories. For directories, the directory's
own `mtime` is used for ordering, and the directory is removed as a whole.

Limitation:

The service does not currently delete just enough items to land exactly on the
configured limit.
