import { SVGProps } from "react";

export function PdfIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      {...props}
    >
      {/* File shape with folded corner */}
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <polyline points="14 2 14 8 20 8" />
      {/* PDF text */}
      <text
        x="12"
        y="16"
        textAnchor="middle"
        fontSize="6"
        fontWeight="bold"
        fill="currentColor"
        stroke="none"
        fontFamily="system-ui, sans-serif"
      >
        PDF
      </text>
    </svg>
  );
}
