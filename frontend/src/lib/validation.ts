import { z } from "zod";

export const positiveIntSchema = z.number().int().nonnegative();
export const upcomingRenewalsWindowSchema = z.number().int().positive().max(3650).default(90);
export const offsetSchema = z.number().int().nonnegative().default(0);
