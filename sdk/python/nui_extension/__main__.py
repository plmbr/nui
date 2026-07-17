"""Default nui-ext entrypoint — subclass NuiExtension in your package."""

from nui_extension import NuiExtension


class DefaultExtension(NuiExtension):
    pass


def main() -> None:
    DefaultExtension().serve()


if __name__ == "__main__":
    main()
