import { Mailer } from "../domain/ports";

interface Message {
  to: string;
  subject: string;
  body: string;
}

export class InMemoryMailer implements Mailer {
  private readonly outbox: Message[] = [];

  async send(to: string, subject: string, body: string): Promise<void> {
    if (!to.includes("@")) {
      throw new Error("invalid recipient");
    }
    this.outbox.push({ to, subject, body });
  }

  size(): number {
    return this.outbox.length;
  }
}
