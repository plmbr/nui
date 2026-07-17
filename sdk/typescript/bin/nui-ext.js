#!/usr/bin/env node
import { NuiExtension } from "../NuiExtension.js";

class DefaultExtension extends NuiExtension {}

new DefaultExtension().serve();
