import re
import sys
import re
import subprocess


def cli_run(cmd: str):
    with subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE) as p:
        if p.returncode != None and p.returncode != 0:
            raise ValueError(f"'{cmd}' failed, returncode {p.returncode}")
        std = str(p.stdout.read(), 'utf-8')
        return std


def current_branch():
    out = cli_run("git status")
    lines = out.splitlines()
    for l in lines:
        m = re.match('On branch ([^\s]+)', l)
        if m:
            return m[1]

def all_tags():
    return cli_run("git tag")

def current_tag():
    out = cli_run("git describe --tags --abbrev=0")
    return out


def parse_beta(tag):
    pat = re.compile('(v.+).beta.*')
    m = pat.match(tag)
    if m:
        return m[1]
    return tag

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Please specify version")
        exit(1)

    branch = current_branch()
    target = sys.argv[1]

    latest_tag = current_tag().strip()
    if latest_tag == target:
        print(f"{latest_tag} has been released")
        exit(1)

    target_release = parse_beta(target)
    if target_release != target and target_release in all_tags().splitlines():
        print(f"{target_release} has been released")
        exit(1)

    with open("./miso/version.go", "w") as f:
        f.writelines([
            "package miso\n",
            "\n",
            "const (\n",
            f"\tMisoVersion = \"{target}\"\n"
            ")\n"
            ""
        ])

    print(cli_run("go fmt ./..."))
    print(cli_run(f"git commit -am \"Release {target}\""))
    print(cli_run(f"git tag \"{target}\""))
    print("Done, it's time to push your tag to remote origin! :D")
    print(f"\ngit push && git push origin {target}\n\n")
