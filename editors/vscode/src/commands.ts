import * as vscode from "vscode";
import { Client } from "./client";

export function registerCommands(context: vscode.ExtensionContext, client: Client, outputChannel: vscode.OutputChannel, traceOutputChannel: vscode.OutputChannel): vscode.Disposable[] {
    const disposables: vscode.Disposable[] = [];

    disposables.push(vscode.commands.registerCommand("golar.restart", () => {
        return client.restart(context);
    }));

    disposables.push(vscode.commands.registerCommand("golar.output.focus", () => {
        outputChannel.show();
    }));

    disposables.push(vscode.commands.registerCommand("golar.lsp-trace.focus", () => {
        traceOutputChannel.show();
    }));

    disposables.push(vscode.commands.registerCommand("golar.showMenu", showCommands));

    disposables.push(vscode.commands.registerCommand("golar.reportIssue", () => {
        vscode.commands.executeCommand("workbench.action.openIssueReporter", {
            extensionId: "nonfx.golar",
        });
    }));

    return disposables;
}

async function showCommands(): Promise<void> {
    const commands: readonly { label: string; description: string; command: string; }[] = [
        {
            label: "$(refresh) Restart Server",
            description: "Restart the Golar language server",
            command: "golar.restart",
        },
        {
            label: "$(output) Show Server Log",
            description: "Show the Golar server log",
            command: "golar.output.focus",
        },
        {
            label: "$(debug-console) Show LSP Messages",
            description: "Show the LSP communication trace",
            command: "golar.lsp-trace.focus",
        },
        {
            label: "$(report) Report Issue",
            description: "Report an issue with Golar",
            command: "golar.reportIssue",
        },
    ];

    const selected = await vscode.window.showQuickPick(commands, {
        placeHolder: "Golar Commands",
    });

    if (selected) {
        await vscode.commands.executeCommand(selected.command);
    }
}
