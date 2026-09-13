"use client";

import { useEffect, useState } from "react";
import { api } from "@/services/api";

export default function BrandLogo({logoDataURL,className=""}:{logoDataURL?:string;className?:string}){
  const [fetchedLogo,setFetchedLogo]=useState("");
  useEffect(()=>{if(logoDataURL===undefined)api.siteSettings().then(settings=>setFetchedLogo(settings.logo_data_url)).catch(()=>undefined)},[logoDataURL]);
  const resolvedLogo=logoDataURL??fetchedLogo;
  if(!resolvedLogo) return null;
  return <span className={`logo brand-logo has-image ${className}`.trim()} style={{backgroundImage:`url(${resolvedLogo})`}} aria-label="Logo platform" role="img"/>;
}
