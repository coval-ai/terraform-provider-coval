import path from "node:path";
import { fileURLToPath } from "node:url";
import { validateReleaseContract } from "./release-contract.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
process.chdir(root);
const args = process.argv.slice(2);
if (args.length > 1 || (args.length === 1 && args[0] !== "--dry-run")) {
  throw new Error("Only --dry-run is supported; release configuration must come from the reviewed repository contract");
}
const config = await validateReleaseContract(root);
// Import after validation so configuration failures cannot reach release hooks.
const { default: semanticRelease } = await import("semantic-release");
await semanticRelease({ ...config, dryRun: args[0] === "--dry-run" }, { cwd: root });
