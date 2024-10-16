# Nuke It From Orbit

*It's the only way to be sure.*

**tl;dr: unprivileged user -> Defender removal on physical machine**

With a precision of a brain surgeon wielding a chainsaw, nifo can obliterate most AV/EDR products from endpoints or servers running the worlds most popular operating system, even if they're BitLocker protected - if you have physical access to the device and it's not totally locked down (BIOS password + SecureBoot + Harddrive Password + No USB Boot).

## How does it work?

Since security on Windows is an afterthought, the operating system can manage quite fine without any AV/EDR software installed. So if you want to disable AV/EDR, it's just a question of breaking it enough for it not to start up.

While protections might be in place to prevent you from tampering directly with registry keys or files from inside the OS, nifo takes the direct approach and overwrites the first bytes of the target files while the operating system is not running - either by booting to Linux via USB or removing the harddrive and putting it in another system to do the same modifications.

It makes no difference if the machine is BitLocker protected or not, since the task is not to write anything particular, just to corrupt the files enough for them not to load when booting. Nifty, I might say.

## Supported (LOL) AV/EDR products

- Microsoft Defender

It's easy to plug in more products as a detector in the code - pull requests for others are welcome, if you have the paths to look for the executables and drivers. It's beyond me to do this for every single product out there.

## Usage

You're not elevated, and want a bash script that will nifo the AV/EDR when booting from another media:

```
nifo.exe generate --method bash [--relativeto drive] > nifo.sh
```

If you're elevated, you can use the `--relativeto drive' to get device offsets rather than partition offset.

Either boot on the target device or on your own machine with the harddrive/SSD connected to a Linux system, and run the script:

```
# chmod +x nifo.sh
# ./nifo.sh nuke /dev/[device/partition]
```

It's up to you to run the generated script with the *correct* device or partition name (/dev/sda, /dev/sda2, /dev/nvme0n1, /dev/nvme0n1p2 etc). Under Linux you can use `blkid` or `fdisk -l /dev/sda` to find out what's what.

Potential recovery with:

```
# ./nifo.sh puke /dev/[device/partition]
```

... but don't hold your breath. It might not work as you expect.

## Other command line options

List supported (LOL) AV/EDR products:

```
nifo.exe products
```

Pull requests accepted for other products - quite easy to add in the code.

## Protecting yourself from this

- Secure BIOS/UEFI settings with a password
- Secure harddrive/SSD data with BitLocker (otherwise you could just scan the drive for the sectors)
- Secure harddrive/SSD from foreign modifications with harddrive password
- Disallow booting off anything but the harddrive.

Fail either of those, and you can get nifo'ed.

## Disclaimer

THERE ARE ABSOLUTELY NO WARRANTIES, GUARANTEES OR OTHER KINDS OF PROMISES, NEITHER EXPRESSED NOR IMPLIED. THIS WILL DESTROY DATA, GET YOU FIRED, PUT IN JAIL, YOUR CAT WILL LOSE ALL ITS FUR, AND YOU'LL UNLEASH THE UNDERWORLD. Don't run this!
