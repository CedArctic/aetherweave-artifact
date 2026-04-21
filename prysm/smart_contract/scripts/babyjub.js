import { buildBabyjub } from "circomlibjs";

const readStdin = async () => {
  const chunks = [];
  for await (const c of process.stdin) chunks.push(c);
  return Buffer.concat(chunks).toString();
};

(async () => {
  const req = JSON.parse(await readStdin());
  const babyjub = await buildBabyjub();
  const L = babyjub.subOrder;

  let sk = BigInt(req.sk ?? "0") % L;
  if (sk === 0n) sk = 1n;

  const [x, y] = babyjub.mulPointEscalar(babyjub.Base8, sk);
  console.log(
    JSON.stringify(
      {
        ok: true,
        x: babyjub.F.toObject(x).toString(),
        y: babyjub.F.toObject(y).toString()
      }
    )
  );
})().catch(e => {
  console.error(e);
  console.log(JSON.stringify({ ok: false, error: String(e) }));
  process.exit(1);
});
