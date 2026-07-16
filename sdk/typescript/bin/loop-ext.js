#!/usr/bin/env node
import { LoopExtension } from "../LoopExtension.js";

class DefaultExtension extends LoopExtension {}

new DefaultExtension().serve();
