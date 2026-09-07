"use client";
import { Fragment } from "react";
import { InlineMath } from "react-katex";

export default function RichText({content}:{content:string}) {
  return <>{content.split(/(\$[^$]+\$)/g).filter(Boolean).map((part,index)=><Fragment key={index}>{part.startsWith("$")&&part.endsWith("$")?<InlineMath math={part.slice(1,-1)}/>:part}</Fragment>)}</>;
}
