load("github.com/cirrus-modules/golang", "lint_task")

def main(ctx):
    return [lint_task()]
