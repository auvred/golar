import * as path from "path";
import * as vscode from "vscode";
import packageJson from "../package.json";

const version = packageJson.version;

export const jsTsLanguageModes = [
    "typescript",
    "typescriptreact",
    "javascript",
    "javascriptreact",
    "vue",
];

export interface ExeInfo {
    path: string;
    version: string;
}

export function getBuiltinExePath(context: vscode.ExtensionContext): string {
    return context.asAbsolutePath(path.join("./lib", `tsgo${process.platform === "win32" ? ".exe" : ""}`));
}

function workspaceResolve(relativePath: string): vscode.Uri {
    if (path.isAbsolute(relativePath)) {
        return vscode.Uri.file(relativePath);
    }
    if (vscode.workspace.workspaceFolders && vscode.workspace.workspaceFolders.length > 0) {
        const workspaceFolder = vscode.workspace.workspaceFolders[0];
        return vscode.Uri.joinPath(workspaceFolder.uri, relativePath);
    }
    return vscode.Uri.file(relativePath);
}

export async function getExe(context: vscode.ExtensionContext): Promise<ExeInfo> {
    const config = vscode.workspace.getConfiguration("golar");
    const exeName = `tsgo${process.platform === "win32" ? ".exe" : ""}`;

    let exe = config.get<string>("tsdk");
    if (exe) {
        try {
            const exePath = workspaceResolve(path.join(exe, exeName));
            await vscode.workspace.fs.stat(exePath);
            return { path: exePath.fsPath, version: "(local)" };
        }
        catch {}
    }

    return {
        path: getBuiltinExePath(context),
        version,
    };
}

export function getLanguageForUri(uri: vscode.Uri): string | undefined {
    const ext = path.posix.extname(uri.path);
    switch (ext) {
        case ".ts":
        case ".mts":
        case ".cts":
            return "typescript";
        case ".js":
        case ".mjs":
        case ".cjs":
            return "javascript";
        case ".tsx":
            return "typescriptreact";
        case ".jsx":
            return "javascriptreact";
        case ".vue":
            return "vue";
        default:
            return undefined;
    }
}
