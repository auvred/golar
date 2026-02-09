import * as vscode from "vscode";

import { Client } from "./client";
import { registerCommands } from "./commands";
import { setupStatusBar } from "./statusBar";
import { setupVersionStatusItem } from "./versionStatusItem";

export async function activate(context: vscode.ExtensionContext) {
    const output = vscode.window.createOutputChannel("golar", { log: true });
    const traceOutput = vscode.window.createOutputChannel("golar (LSP)", { log: true });
    context.subscriptions.push(output, traceOutput);

    const disposable = await activateLanguageFeatures(context, output, traceOutput);
    context.subscriptions.push(disposable);
}

async function activateLanguageFeatures(context: vscode.ExtensionContext, output: vscode.LogOutputChannel, traceOutput: vscode.LogOutputChannel): Promise<vscode.Disposable> {
    const disposables: vscode.Disposable[] = [];

    const client = new Client(output, traceOutput);
    disposables.push(...registerCommands(context, client, output, traceOutput));
    disposables.push(await client.initialize(context));
    disposables.push(setupStatusBar());
    disposables.push(...setupVersionStatusItem(client));
    return vscode.Disposable.from(...disposables);
}
