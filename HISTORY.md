# GoPOSIX History

## Why GoPOSIX Exists ?

Well, I wanted to do an experiment on [Harsness Engineering](https://walkinglabs.github.io/learn-harness-engineering/en/), and improve my "agentic development" skills, prompts, instructions and all that.

I did [LFS](https://www.linuxfromscratch.org/) in my early 20's and I had this weird itch of "do your own toolsets" for some reason, but left it alone for my own sanity and lack of technical expertise. Still the whole POSIX thinking model and concepts never left the the back of my head.

Last year (2025) I started to learn Go-lang, and then LLMs got *really good* in December 2025. Good enough that I've been using it at work non stop since then.

During that time I got this notion that AI waste time formating output (grep/sed/awk is a constant still today (Oct 2026)), so I started doing `--json` output in a lot of my work scripts and tools to save me some tokens and time for my robot friends.

Eventually... all of these random ideas boiled to the conclusion that **I should** make a complete implementation of POSIX utilities in Go, with a JSON output and then benchmark it against BusyBox. 

It's kind of the "natural conclusion" ... right?????

(Also deepseek-v4-pro had an very agressive [75% discount](https://api-docs.deepseek.com/quick_start/pricing)!, and I wanted to try [pi.dev](https://pi.dev) instead of Antigravity/ClaudeCode (I ended up using `agy` for some auditing))

All things kind of aligned in the last month (May 2025) so here we are now.

I'm not the first to start something like this, there is [cugo](https://github.com/jcmdln/cugo) and [go-posix](https://github.com/nirenjan/go-posix), but sadly they seem to be abandoned, and no wonder! A project like this is a huge undertaking, its probably a year of solid work for 1 human, that being said, took about 3 weeks to do with AI, with the proper "harness" and "agentic development" approach, that's really something.

Anyway the project got into a point that _I'm happy_ with the results, that's enough for me ☑️.

## Honest and Obvious Recognitions

I want to be very clear about this:
>  The only reason this works is that there's a brutally thorough, existing corpus of tests to validate against. The AI iterated until it passed. Without BusyBox's tests, this project is just random hallucinated code. **The test suite is the real hero**.

- So check and support [BusyBox](https://busybox.net/) project and take a look at its amazing [test suite](https://github.com/brgl/busybox/blob/master/testsuite/runtest), it's a masterpiece of thoroughness and coverage, and it made this project possible.
- Also [Mvdan Shell](https://github.com/mvdan/sh), it really saved my butt. Absolutely brilliant.
- And [goawk](https://github.com/benhoyt/goawk), which I used for the `awk` implementation, another big save!
- [blakesmith/ar](https://github.com/blakesmith/ar) for reading and writing `ar` archives.
- [cavaliergopher/cpio](https://github.com/cavaliergopher/cpio) for `cpio` archive support.
- [hotei/dcompress](https://github.com/hotei/dcompress) for decompression helpers.
- [sergeymakinen/go-crypt](https://github.com/sergeymakinen/go-crypt) and [tredoe/crypt](https://github.com/tredoe/crypt) for unix password crypt implementation.
- [ulikunitz/xz](https://github.com/ulikunitz/xz) for xz/lzma compression and decompression.
- `golang.org/x` libraries ([sys](https://pkg.go.dev/golang.org/x/sys), [term](https://pkg.go.dev/golang.org/x/term), [crypto](https://pkg.go.dev/golang.org/x/crypto)) for system call access, terminal control, and cryptographic functions.

Finally: let's not kid ourselves, this project is 90% wiring the AI to do the heavy lifting, 10% is steering it in the right direction, the fact that I was able to "solo dev" this with an LLM, reproducing close to 99% of BusyBox's behavior in a completely different language shows that POSIX utilities are, at their core, text transformers with very well-defined contracts (do one thing and do it well).

Thanks for reading the story, and I hope you enjoy the project as much as I enjoyed building it.

With craft and harness, @ramayac.
