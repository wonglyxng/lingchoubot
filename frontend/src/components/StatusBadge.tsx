"use client";

import { getVariantClass, type StatusVariant } from "@/lib/utils";

export function StatusBadge({
  label,
  variant,
}: {
  label: string;
  variant: StatusVariant;
}) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${getVariantClass(variant)}`}
    >
      {label}
    </span>
  );
}
