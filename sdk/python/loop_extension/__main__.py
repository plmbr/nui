"""Default loop-ext entrypoint — subclass LoopExtension in your package."""

from loop_extension import LoopExtension


class DefaultExtension(LoopExtension):
    pass


def main() -> None:
    DefaultExtension().serve()


if __name__ == "__main__":
    main()
