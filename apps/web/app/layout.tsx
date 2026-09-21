import type { Metadata } from "next";
import { Geist_Mono, Inter, Outfit } from "next/font/google";
import "./globals.css";

// Three voices: Inter for the interface, Outfit for the wordmark,
// Geist Mono for the archival stamps (time, domains, counts).
const ui = Inter({
  subsets: ["latin"],
  variable: "--font-ui",
});

const mark = Outfit({
  subsets: ["latin"],
  variable: "--font-mark",
});

const geist = Geist_Mono({
  subsets: ["latin"],
  variable: "--font-geist",
});

export const metadata: Metadata = {
  title: "helmme — personal memory",
  description: "Save anything. Organize nothing. Find everything.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${ui.variable} ${mark.variable} ${geist.variable}`}>
      <body className="bg-light font-sans text-deep antialiased">{children}</body>
    </html>
  );
}
