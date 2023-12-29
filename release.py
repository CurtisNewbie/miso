import sys
import re
import subprocess


def cli_run(cmd: str):
    with subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE) as p:
        std = str(p.stdout.read(), 'utf-8')
        return std


def current_branch():
    out = cli_run("git status")
    lines = out.splitlines()
    for l in lines:
        m = re.match('On branch ([^\s]+)', l)
        if m:
            return m[1]


def current_tag():
    out = cli_run("git describe --tags --abbrev=0")
    return out


if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Please specify version")
        exit(1)

    branch = current_branch()
    print(branch)

    if branch == 'dev':
        print("Cannot release on dev branch")
        exit(1)

    release = sys.argv[1]
    latest = current_tag()
    if latest == release:
        print(f"{release} has been released")
        exit(1)

    with open("./miso/version.go") as f:
        f.writelines([
            "package miso",
            "",
            "const (",
            f"\tMisoVersion = \"{release}\""
            ")"
            ""
        ])

    print(cli_run("go fmt ./..."))
    print(cli_run(f"git commit -am \"Release {release}\""))
    print(cli_run(f"git tag \"{release}\""))
    print("Done, it's time to push your tag to remote origin! :D")
    print(f"\ngit push && git push origin {release}\n\n")
