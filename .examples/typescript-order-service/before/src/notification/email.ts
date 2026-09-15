const OUTBOX: Array<{ to: string; subject: string; body: string }> = [];

export async function sendEmail(to: string, subject: string, body: string): Promise<void> {
  if (!to.includes("@")) {
    throw new Error("invalid recipient");
  }
  OUTBOX.push({ to, subject, body });
}

export function outboxSize(): number {
  return OUTBOX.length;
}
