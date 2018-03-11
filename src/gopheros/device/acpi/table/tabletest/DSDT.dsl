/*
 * Intel ACPI Component Architecture
 * AML/ASL+ Disassembler version 20160108-64
 * Copyright (c) 2000 - 2016 Intel Corporation
 * 
 * Disassembling to symbolic ASL+ operators
 *
 * Disassembly of DSDT.aml, Thu Jan  4 07:55:57 2018
 *
 * Original Table Header:
 *     Signature        "DSDT"
 *     Length           0x000021C8 (8648)
 *     Revision         0x02
 *     Checksum         0xEE
 *     OEM ID           "VBOX  "
 *     OEM Table ID     "VBOXBIOS"
 *     OEM Revision     0x00000002 (2)
 *     Compiler ID      "INTL"
 *     Compiler Version 0x20100528 (537920808)
 */
DefinitionBlock ("DSDT.aml", "DSDT", 2, "VBOX  ", "VBOXBIOS", 0x00000002)
{
    OperationRegion (DBG0, SystemIO, 0x3000, 0x04)
    Field (DBG0, ByteAcc, NoLock, Preserve)
    {
        DHE1,   8
    }

    Field (DBG0, WordAcc, NoLock, Preserve)
    {
        DHE2,   16
    }

    Field (DBG0, DWordAcc, NoLock, Preserve)
    {
        DHE4,   32
    }

    Field (DBG0, ByteAcc, NoLock, Preserve)
    {
        Offset (0x01), 
        DCHR,   8
    }

    Method (HEX, 1, NotSerialized)
    {
        DHE1 = Arg0
    }

    Method (HEX2, 1, NotSerialized)
    {
        DHE2 = Arg0
    }

    Method (HEX4, 1, NotSerialized)
    {
        DHE4 = Arg0
    }

    Method (SLEN, 1, NotSerialized)
    {
        Local0 = Arg0
        Return (SizeOf (Local0))
    }

    Method (S2BF, 1, Serialized)
    {
        Local0 = Arg0
        Local0 = (SLEN (Local0) + One)
        Name (BUFF, Buffer (Local0) {})
        BUFF = Arg0
        Return (BUFF) /* \S2BF.BUFF */
    }

    Method (MIN, 2, NotSerialized)
    {
        If ((Arg0 < Arg1))
        {
            Return (Arg0)
        }
        Else
        {
            Return (Arg1)
        }
    }

    Method (SCMP, 2, NotSerialized)
    {
        Local0 = Arg0
        Local0 = S2BF (Local0)
        Local1 = S2BF (Arg1)
        Local4 = Zero
        Local5 = SLEN (Arg0)
        Local6 = SLEN (Arg1)
        Local7 = MIN (Local5, Local6)
        While ((Local4 < Local7))
        {
            Local2 = DerefOf (Local0 [Local4])
            Local3 = DerefOf (Local1 [Local4])
            If ((Local2 > Local3))
            {
                Return (One)
            }
            ElseIf ((Local2 < Local3))
            {
                Return (Ones)
            }

            Local4++
        }

        If ((Local4 < Local5))
        {
            Return (One)
        }
        ElseIf ((Local4 < Local6))
        {
            Return (Ones)
        }
        Else
        {
            Return (Zero)
        }
    }

    Method (MTCH, 2, NotSerialized)
    {
        Local0 = Arg0
        Local1 = Arg1
        Local2 = SCMP (Local0, Local1)
        Return (!Local2)
    }

    Method (DBG, 1, NotSerialized)
    {
        Local0 = Arg0
        Local1 = S2BF (Local0)
        Local0 = SizeOf (Local1)
        Local0--
        Local2 = Zero
        While (Local0)
        {
            Local0--
            DCHR = DerefOf (Local1 [Local2])
            Local2++
        }
    }

    Name (MSWV, Ones)
    Method (MSWN, 0, NotSerialized)
    {
        If ((MSWV != Ones))
        {
            Return (MSWV) /* \MSWV */
        }

        MSWV = Zero
        DBG ("_OS: ")
        DBG (_OS)
        DBG ("\n")
        If (CondRefOf (_OSI))
        {
            DBG ("_OSI exists\n")
            If (_OSI ("Windows 2001"))
            {
                MSWV = 0x04
            }

            If (_OSI ("Windows 2001.1"))
            {
                MSWV = 0x05
            }

            If (_OSI ("Windows 2006"))
            {
                MSWV = 0x06
            }

            If (_OSI ("Windows 2009"))
            {
                MSWV = 0x07
            }

            If (_OSI ("Windows 2012"))
            {
                MSWV = 0x08
            }

            If (_OSI ("Windows 2013"))
            {
                MSWV = 0x09
            }

            If (_OSI ("Windows 2015"))
            {
                MSWV = 0x0A
            }

            If (_OSI ("Windows 2006 SP2"))
            {
                DBG ("Windows 2006 SP2 supported\n")
                MSWV = Zero
            }
        }
        ElseIf (MTCH (_OS, "Microsoft Windows NT"))
        {
            MSWV = 0x03
        }

        If (CondRefOf (_REV))
        {
            DBG ("_REV: ")
            HEX4 (_REV)
            If (((MSWV > Zero) && (_REV > 0x02)))
            {
                If ((MSWV < 0x08))
                {
                    DBG ("ACPI rev mismatch, not a Microsoft OS\n")
                    MSWV = Zero
                }
            }
        }

        DBG ("Determined MSWV: ")
        HEX4 (MSWV)
        Return (MSWV) /* \MSWV */
    }

    Name (PICM, Zero)
    Method (_PIC, 1, NotSerialized)  // _PIC: Interrupt Model
    {
        DBG ("Pic mode: ")
        HEX4 (Arg0)
        PICM = Arg0
    }

    OperationRegion (SYSI, SystemIO, 0x4048, 0x08)
    Field (SYSI, DWordAcc, NoLock, Preserve)
    {
        IDX0,   32, 
        DAT0,   32
    }

    IndexField (IDX0, DAT0, DWordAcc, NoLock, Preserve)
    {
        MEML,   32, 
        UIOA,   32, 
        UHPT,   32, 
        USMC,   32, 
        UFDC,   32, 
        SL2B,   32, 
        SL2I,   32, 
        SL3B,   32, 
        SL3I,   32, 
        PMNN,   32, 
        URTC,   32, 
        CPUL,   32, 
        CPUC,   32, 
        CPET,   32, 
        CPEV,   32, 
        NICA,   32, 
        HDAA,   32, 
        PWRS,   32, 
        IOCA,   32, 
        HBCA,   32, 
        PCIB,   32, 
        PCIL,   32, 
        SL0B,   32, 
        SL0I,   32, 
        SL1B,   32, 
        SL1I,   32, 
        PP0B,   32, 
        PP0I,   32, 
        PP1B,   32, 
        PP1I,   32, 
        PMNX,   32, 
        Offset (0x80), 
        ININ,   32, 
        Offset (0x200), 
        VAIN,   32
    }

    Scope (_SB)
    {
        Method (_INI, 0, NotSerialized)  // _INI: Initialize
        {
            VAIN = 0x0BADC0DE
            DBG ("MEML: ")
            HEX4 (MEML)
            DBG ("UIOA: ")
            HEX4 (UIOA)
            DBG ("UHPT: ")
            HEX4 (UHPT)
            DBG ("USMC: ")
            HEX4 (USMC)
            DBG ("UFDC: ")
            HEX4 (UFDC)
            DBG ("PMNN: ")
            HEX4 (PMNN)
        }

        Name (PR00, Package (0x78)
        {
            Package (0x04)
            {
                0x0002FFFF, 
                Zero, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0002FFFF, 
                One, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0002FFFF, 
                0x02, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x0002FFFF, 
                0x03, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0003FFFF, 
                Zero, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0003FFFF, 
                One, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x0003FFFF, 
                0x02, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0003FFFF, 
                0x03, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0004FFFF, 
                Zero, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x0004FFFF, 
                One, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0004FFFF, 
                0x02, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0004FFFF, 
                0x03, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0005FFFF, 
                Zero, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0005FFFF, 
                One, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0005FFFF, 
                0x02, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0005FFFF, 
                0x03, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x0006FFFF, 
                Zero, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0006FFFF, 
                One, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0006FFFF, 
                0x02, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x0006FFFF, 
                0x03, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0007FFFF, 
                Zero, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0007FFFF, 
                One, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x0007FFFF, 
                0x02, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0007FFFF, 
                0x03, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0008FFFF, 
                Zero, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x0008FFFF, 
                One, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0008FFFF, 
                0x02, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0008FFFF, 
                0x03, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0009FFFF, 
                Zero, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0009FFFF, 
                One, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0009FFFF, 
                0x02, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0009FFFF, 
                0x03, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x000AFFFF, 
                Zero, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x000AFFFF, 
                One, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x000AFFFF, 
                0x02, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x000AFFFF, 
                0x03, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x000BFFFF, 
                Zero, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x000BFFFF, 
                One, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x000BFFFF, 
                0x02, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x000BFFFF, 
                0x03, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x000CFFFF, 
                Zero, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x000CFFFF, 
                One, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x000CFFFF, 
                0x02, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x000CFFFF, 
                0x03, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x000DFFFF, 
                Zero, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x000DFFFF, 
                One, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x000DFFFF, 
                0x02, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x000DFFFF, 
                0x03, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x000EFFFF, 
                Zero, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x000EFFFF, 
                One, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x000EFFFF, 
                0x02, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x000EFFFF, 
                0x03, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x000FFFFF, 
                Zero, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x000FFFFF, 
                One, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x000FFFFF, 
                0x02, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x000FFFFF, 
                0x03, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0010FFFF, 
                Zero, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x0010FFFF, 
                One, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0010FFFF, 
                0x02, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0010FFFF, 
                0x03, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0011FFFF, 
                Zero, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0011FFFF, 
                One, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0011FFFF, 
                0x02, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0011FFFF, 
                0x03, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x0012FFFF, 
                Zero, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0012FFFF, 
                One, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0012FFFF, 
                0x02, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x0012FFFF, 
                0x03, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0013FFFF, 
                Zero, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0013FFFF, 
                One, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x0013FFFF, 
                0x02, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0013FFFF, 
                0x03, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0014FFFF, 
                Zero, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x0014FFFF, 
                One, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0014FFFF, 
                0x02, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0014FFFF, 
                0x03, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0015FFFF, 
                Zero, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0015FFFF, 
                One, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0015FFFF, 
                0x02, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0015FFFF, 
                0x03, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x0016FFFF, 
                Zero, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0016FFFF, 
                One, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0016FFFF, 
                0x02, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x0016FFFF, 
                0x03, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0017FFFF, 
                Zero, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0017FFFF, 
                One, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x0017FFFF, 
                0x02, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0017FFFF, 
                0x03, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0018FFFF, 
                Zero, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x0018FFFF, 
                One, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0018FFFF, 
                0x02, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0018FFFF, 
                0x03, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0019FFFF, 
                Zero, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x0019FFFF, 
                One, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x0019FFFF, 
                0x02, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x0019FFFF, 
                0x03, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x001AFFFF, 
                Zero, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x001AFFFF, 
                One, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x001AFFFF, 
                0x02, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x001AFFFF, 
                0x03, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x001BFFFF, 
                Zero, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x001BFFFF, 
                One, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x001BFFFF, 
                0x02, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x001BFFFF, 
                0x03, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x001CFFFF, 
                Zero, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x001CFFFF, 
                One, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x001CFFFF, 
                0x02, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x001CFFFF, 
                0x03, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x001DFFFF, 
                Zero, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x001DFFFF, 
                One, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x001DFFFF, 
                0x02, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x001DFFFF, 
                0x03, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x001EFFFF, 
                Zero, 
                LNKB, 
                Zero
            }, 

            Package (0x04)
            {
                0x001EFFFF, 
                One, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x001EFFFF, 
                0x02, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x001EFFFF, 
                0x03, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x001FFFFF, 
                Zero, 
                LNKC, 
                Zero
            }, 

            Package (0x04)
            {
                0x001FFFFF, 
                One, 
                LNKD, 
                Zero
            }, 

            Package (0x04)
            {
                0x001FFFFF, 
                0x02, 
                LNKA, 
                Zero
            }, 

            Package (0x04)
            {
                0x001FFFFF, 
                0x03, 
                LNKB, 
                Zero
            }
        })
        Name (PR01, Package (0x78)
        {
            Package (0x04)
            {
                0x0002FFFF, 
                Zero, 
                Zero, 
                0x12
            }, 

            Package (0x04)
            {
                0x0002FFFF, 
                One, 
                Zero, 
                0x13
            }, 

            Package (0x04)
            {
                0x0002FFFF, 
                0x02, 
                Zero, 
                0x14
            }, 

            Package (0x04)
            {
                0x0002FFFF, 
                0x03, 
                Zero, 
                0x15
            }, 

            Package (0x04)
            {
                0x0003FFFF, 
                Zero, 
                Zero, 
                0x13
            }, 

            Package (0x04)
            {
                0x0003FFFF, 
                One, 
                Zero, 
                0x14
            }, 

            Package (0x04)
            {
                0x0003FFFF, 
                0x02, 
                Zero, 
                0x15
            }, 

            Package (0x04)
            {
                0x0003FFFF, 
                0x03, 
                Zero, 
                0x16
            }, 

            Package (0x04)
            {
                0x0004FFFF, 
                Zero, 
                Zero, 
                0x14
            }, 

            Package (0x04)
            {
                0x0004FFFF, 
                One, 
                Zero, 
                0x15
            }, 

            Package (0x04)
            {
                0x0004FFFF, 
                0x02, 
                Zero, 
                0x16
            }, 

            Package (0x04)
            {
                0x0004FFFF, 
                0x03, 
                Zero, 
                0x17
            }, 

            Package (0x04)
            {
                0x0005FFFF, 
                Zero, 
                Zero, 
                0x15
            }, 

            Package (0x04)
            {
                0x0005FFFF, 
                One, 
                Zero, 
                0x16
            }, 

            Package (0x04)
            {
                0x0005FFFF, 
                0x02, 
                Zero, 
                0x17
            }, 

            Package (0x04)
            {
                0x0005FFFF, 
                0x03, 
                Zero, 
                0x10
            }, 

            Package (0x04)
            {
                0x0006FFFF, 
                Zero, 
                Zero, 
                0x16
            }, 

            Package (0x04)
            {
                0x0006FFFF, 
                One, 
                Zero, 
                0x17
            }, 

            Package (0x04)
            {
                0x0006FFFF, 
                0x02, 
                Zero, 
                0x10
            }, 

            Package (0x04)
            {
                0x0006FFFF, 
                0x03, 
                Zero, 
                0x11
            }, 

            Package (0x04)
            {
                0x0007FFFF, 
                Zero, 
                Zero, 
                0x17
            }, 

            Package (0x04)
            {
                0x0007FFFF, 
                One, 
                Zero, 
                0x10
            }, 

            Package (0x04)
            {
                0x0007FFFF, 
                0x02, 
                Zero, 
                0x11
            }, 

            Package (0x04)
            {
                0x0007FFFF, 
                0x03, 
                Zero, 
                0x12
            }, 

            Package (0x04)
            {
                0x0008FFFF, 
                Zero, 
                Zero, 
                0x10
            }, 

            Package (0x04)
            {
                0x0008FFFF, 
                One, 
                Zero, 
                0x11
            }, 

            Package (0x04)
            {
                0x0008FFFF, 
                0x02, 
                Zero, 
                0x12
            }, 

            Package (0x04)
            {
                0x0008FFFF, 
                0x03, 
                Zero, 
                0x13
            }, 

            Package (0x04)
            {
                0x0009FFFF, 
                Zero, 
                Zero, 
                0x11
            }, 

            Package (0x04)
            {
                0x0009FFFF, 
                One, 
                Zero, 
                0x12
            }, 

            Package (0x04)
            {
                0x0009FFFF, 
                0x02, 
                Zero, 
                0x13
            }, 

            Package (0x04)
            {
                0x0009FFFF, 
                0x03, 
                Zero, 
                0x14
            }, 

            Package (0x04)
            {
                0x000AFFFF, 
                Zero, 
                Zero, 
                0x12
            }, 

            Package (0x04)
            {
                0x000AFFFF, 
                One, 
                Zero, 
                0x13
            }, 

            Package (0x04)
            {
                0x000AFFFF, 
                0x02, 
                Zero, 
                0x14
            }, 

            Package (0x04)
            {
                0x000AFFFF, 
                0x03, 
                Zero, 
                0x15
            }, 

            Package (0x04)
            {
                0x000BFFFF, 
                Zero, 
                Zero, 
                0x13
            }, 

            Package (0x04)
            {
                0x000BFFFF, 
                One, 
                Zero, 
                0x14
            }, 

            Package (0x04)
            {
                0x000BFFFF, 
                0x02, 
                Zero, 
                0x15
            }, 

            Package (0x04)
            {
                0x000BFFFF, 
                0x03, 
                Zero, 
                0x16
            }, 

            Package (0x04)
            {
                0x000CFFFF, 
                Zero, 
                Zero, 
                0x14
            }, 

            Package (0x04)
            {
                0x000CFFFF, 
                One, 
                Zero, 
                0x15
            }, 

            Package (0x04)
            {
                0x000CFFFF, 
                0x02, 
                Zero, 
                0x16
            }, 

            Package (0x04)
            {
                0x000CFFFF, 
                0x03, 
                Zero, 
                0x17
            }, 

            Package (0x04)
            {
                0x000DFFFF, 
                Zero, 
                Zero, 
                0x15
            }, 

            Package (0x04)
            {
                0x000DFFFF, 
                One, 
                Zero, 
                0x16
            }, 

            Package (0x04)
            {
                0x000DFFFF, 
                0x02, 
                Zero, 
                0x17
            }, 

            Package (0x04)
            {
                0x000DFFFF, 
                0x03, 
                Zero, 
                0x10
            }, 

            Package (0x04)
            {
                0x000EFFFF, 
                Zero, 
                Zero, 
                0x16
            }, 

            Package (0x04)
            {
                0x000EFFFF, 
                One, 
                Zero, 
                0x17
            }, 

            Package (0x04)
            {
                0x000EFFFF, 
                0x02, 
                Zero, 
                0x10
            }, 

            Package (0x04)
            {
                0x000EFFFF, 
                0x03, 
                Zero, 
                0x11
            }, 

            Package (0x04)
            {
                0x000FFFFF, 
                Zero, 
                Zero, 
                0x17
            }, 

            Package (0x04)
            {
                0x000FFFFF, 
                One, 
                Zero, 
                0x10
            }, 

            Package (0x04)
            {
                0x000FFFFF, 
                0x02, 
                Zero, 
                0x11
            }, 

            Package (0x04)
            {
                0x000FFFFF, 
                0x03, 
                Zero, 
                0x12
            }, 

            Package (0x04)
            {
                0x0010FFFF, 
                Zero, 
                Zero, 
                0x10
            }, 

            Package (0x04)
            {
                0x0010FFFF, 
                One, 
                Zero, 
                0x11
            }, 

            Package (0x04)
            {
                0x0010FFFF, 
                0x02, 
                Zero, 
                0x12
            }, 

            Package (0x04)
            {
                0x0010FFFF, 
                0x03, 
                Zero, 
                0x13
            }, 

            Package (0x04)
            {
                0x0011FFFF, 
                Zero, 
                Zero, 
                0x11
            }, 

            Package (0x04)
            {
                0x0011FFFF, 
                One, 
                Zero, 
                0x12
            }, 

            Package (0x04)
            {
                0x0011FFFF, 
                0x02, 
                Zero, 
                0x13
            }, 

            Package (0x04)
            {
                0x0011FFFF, 
                0x03, 
                Zero, 
                0x14
            }, 

            Package (0x04)
            {
                0x0012FFFF, 
                Zero, 
                Zero, 
                0x12
            }, 

            Package (0x04)
            {
                0x0012FFFF, 
                One, 
                Zero, 
                0x13
            }, 

            Package (0x04)
            {
                0x0012FFFF, 
                0x02, 
                Zero, 
                0x14
            }, 

            Package (0x04)
            {
                0x0012FFFF, 
                0x03, 
                Zero, 
                0x15
            }, 

            Package (0x04)
            {
                0x0013FFFF, 
                Zero, 
                Zero, 
                0x13
            }, 

            Package (0x04)
            {
                0x0013FFFF, 
                One, 
                Zero, 
                0x14
            }, 

            Package (0x04)
            {
                0x0013FFFF, 
                0x02, 
                Zero, 
                0x15
            }, 

            Package (0x04)
            {
                0x0013FFFF, 
                0x03, 
                Zero, 
                0x16
            }, 

            Package (0x04)
            {
                0x0014FFFF, 
                Zero, 
                Zero, 
                0x14
            }, 

            Package (0x04)
            {
                0x0014FFFF, 
                One, 
                Zero, 
                0x15
            }, 

            Package (0x04)
            {
                0x0014FFFF, 
                0x02, 
                Zero, 
                0x16
            }, 

            Package (0x04)
            {
                0x0014FFFF, 
                0x03, 
                Zero, 
                0x17
            }, 

            Package (0x04)
            {
                0x0015FFFF, 
                Zero, 
                Zero, 
                0x15
            }, 

            Package (0x04)
            {
                0x0015FFFF, 
                One, 
                Zero, 
                0x16
            }, 

            Package (0x04)
            {
                0x0015FFFF, 
                0x02, 
                Zero, 
                0x17
            }, 

            Package (0x04)
            {
                0x0015FFFF, 
                0x03, 
                Zero, 
                0x10
            }, 

            Package (0x04)
            {
                0x0016FFFF, 
                Zero, 
                Zero, 
                0x16
            }, 

            Package (0x04)
            {
                0x0016FFFF, 
                One, 
                Zero, 
                0x17
            }, 

            Package (0x04)
            {
                0x0016FFFF, 
                0x02, 
                Zero, 
                0x10
            }, 

            Package (0x04)
            {
                0x0016FFFF, 
                0x03, 
                Zero, 
                0x11
            }, 

            Package (0x04)
            {
                0x0017FFFF, 
                Zero, 
                Zero, 
                0x17
            }, 

            Package (0x04)
            {
                0x0017FFFF, 
                One, 
                Zero, 
                0x10
            }, 

            Package (0x04)
            {
                0x0017FFFF, 
                0x02, 
                Zero, 
                0x11
            }, 

            Package (0x04)
            {
                0x0017FFFF, 
                0x03, 
                Zero, 
                0x12
            }, 

            Package (0x04)
            {
                0x0018FFFF, 
                Zero, 
                Zero, 
                0x10
            }, 

            Package (0x04)
            {
                0x0018FFFF, 
                One, 
                Zero, 
                0x11
            }, 

            Package (0x04)
            {
                0x0018FFFF, 
                0x02, 
                Zero, 
                0x12
            }, 

            Package (0x04)
            {
                0x0018FFFF, 
                0x03, 
                Zero, 
                0x13
            }, 

            Package (0x04)
            {
                0x0019FFFF, 
                Zero, 
                Zero, 
                0x11
            }, 

            Package (0x04)
            {
                0x0019FFFF, 
                One, 
                Zero, 
                0x12
            }, 

            Package (0x04)
            {
                0x0019FFFF, 
                0x02, 
                Zero, 
                0x13
            }, 

            Package (0x04)
            {
                0x0019FFFF, 
                0x03, 
                Zero, 
                0x14
            }, 

            Package (0x04)
            {
                0x001AFFFF, 
                Zero, 
                Zero, 
                0x12
            }, 

            Package (0x04)
            {
                0x001AFFFF, 
                One, 
                Zero, 
                0x13
            }, 

            Package (0x04)
            {
                0x001AFFFF, 
                0x02, 
                Zero, 
                0x14
            }, 

            Package (0x04)
            {
                0x001AFFFF, 
                0x03, 
                Zero, 
                0x15
            }, 

            Package (0x04)
            {
                0x001BFFFF, 
                Zero, 
                Zero, 
                0x13
            }, 

            Package (0x04)
            {
                0x001BFFFF, 
                One, 
                Zero, 
                0x14
            }, 

            Package (0x04)
            {
                0x001BFFFF, 
                0x02, 
                Zero, 
                0x15
            }, 

            Package (0x04)
            {
                0x001BFFFF, 
                0x03, 
                Zero, 
                0x16
            }, 

            Package (0x04)
            {
                0x001CFFFF, 
                Zero, 
                Zero, 
                0x14
            }, 

            Package (0x04)
            {
                0x001CFFFF, 
                One, 
                Zero, 
                0x15
            }, 

            Package (0x04)
            {
                0x001CFFFF, 
                0x02, 
                Zero, 
                0x16
            }, 

            Package (0x04)
            {
                0x001CFFFF, 
                0x03, 
                Zero, 
                0x17
            }, 

            Package (0x04)
            {
                0x001DFFFF, 
                Zero, 
                Zero, 
                0x15
            }, 

            Package (0x04)
            {
                0x001DFFFF, 
                One, 
                Zero, 
                0x16
            }, 

            Package (0x04)
            {
                0x001DFFFF, 
                0x02, 
                Zero, 
                0x17
            }, 

            Package (0x04)
            {
                0x001DFFFF, 
                0x03, 
                Zero, 
                0x10
            }, 

            Package (0x04)
            {
                0x001EFFFF, 
                Zero, 
                Zero, 
                0x16
            }, 

            Package (0x04)
            {
                0x001EFFFF, 
                One, 
                Zero, 
                0x17
            }, 

            Package (0x04)
            {
                0x001EFFFF, 
                0x02, 
                Zero, 
                0x10
            }, 

            Package (0x04)
            {
                0x001EFFFF, 
                0x03, 
                Zero, 
                0x11
            }, 

            Package (0x04)
            {
                0x001FFFFF, 
                Zero, 
                Zero, 
                0x17
            }, 

            Package (0x04)
            {
                0x001FFFFF, 
                One, 
                Zero, 
                0x10
            }, 

            Package (0x04)
            {
                0x001FFFFF, 
                0x02, 
                Zero, 
                0x11
            }, 

            Package (0x04)
            {
                0x001FFFFF, 
                0x03, 
                Zero, 
                0x12
            }
        })
        Name (PRSA, ResourceTemplate ()
        {
            IRQ (Level, ActiveLow, Shared, )
                {5,9,10,11}
        })
        Name (PRSB, ResourceTemplate ()
        {
            IRQ (Level, ActiveLow, Shared, )
                {5,9,10,11}
        })
        Name (PRSC, ResourceTemplate ()
        {
            IRQ (Level, ActiveLow, Shared, )
                {5,9,10,11}
        })
        Name (PRSD, ResourceTemplate ()
        {
            IRQ (Level, ActiveLow, Shared, )
                {5,9,10,11}
        })
        Device (PCI0)
        {
            Name (_HID, EisaId ("PNP0A03") /* PCI Bus */)  // _HID: Hardware ID
            Method (_ADR, 0, NotSerialized)  // _ADR: Address
            {
                Return (HBCA) /* \HBCA */
            }

            Name (_BBN, Zero)  // _BBN: BIOS Bus Number
            Name (_UID, Zero)  // _UID: Unique ID
            Method (_PRT, 0, NotSerialized)  // _PRT: PCI Routing Table
            {
                If (((PICM && UIOA) == Zero))
                {
                    DBG ("RETURNING PIC\n")
                    ^SBRG.APDE = Zero
                    ^SBRG.APAD = Zero
                    Return (PR00) /* \_SB_.PR00 */
                }
                Else
                {
                    DBG ("RETURNING APIC\n")
                    ^SBRG.APDE = 0xBE
                    ^SBRG.APAD = 0xEF
                    Return (PR01) /* \_SB_.PR01 */
                }
            }

            Device (SBRG)
            {
                Method (_ADR, 0, NotSerialized)  // _ADR: Address
                {
                    Return (IOCA) /* \IOCA */
                }

                OperationRegion (PCIC, PCI_Config, Zero, 0xFF)
                Field (PCIC, ByteAcc, NoLock, Preserve)
                {
                    Offset (0xAD), 
                    APAD,   8, 
                    Offset (0xDE), 
                    APDE,   8
                }

                Device (^PCIE)
                {
                    Name (_HID, EisaId ("PNP0C02") /* PNP Motherboard Resources */)  // _HID: Hardware ID
                    Name (_UID, 0x11)  // _UID: Unique ID
                    Name (CRS, ResourceTemplate ()
                    {
                        Memory32Fixed (ReadOnly,
                            0xDC000000,         // Address Base
                            0x04000000,         // Address Length
                            _Y00)
                    })
                    Method (_CRS, 0, NotSerialized)  // _CRS: Current Resource Settings
                    {
                        CreateDWordField (CRS, \_SB.PCI0.PCIE._Y00._BAS, BAS1)  // _BAS: Base Address
                        CreateDWordField (CRS, \_SB.PCI0.PCIE._Y00._LEN, LEN1)  // _LEN: Length
                        BAS1 = PCIB /* \PCIB */
                        LEN1 = PCIL /* \PCIL */
                        Return (CRS) /* \_SB_.PCI0.PCIE.CRS_ */
                    }

                    Method (_STA, 0, NotSerialized)  // _STA: Status
                    {
                        If ((PCIB == Zero))
                        {
                            Return (Zero)
                        }
                        Else
                        {
                            Return (0x0F)
                        }
                    }
                }

                Device (PS2K)
                {
                    Name (_HID, EisaId ("PNP0303") /* IBM Enhanced Keyboard (101/102-key, PS/2 Mouse) */)  // _HID: Hardware ID
                    Method (_STA, 0, NotSerialized)  // _STA: Status
                    {
                        Return (0x0F)
                    }

                    Name (_CRS, ResourceTemplate ()  // _CRS: Current Resource Settings
                    {
                        IO (Decode16,
                            0x0060,             // Range Minimum
                            0x0060,             // Range Maximum
                            0x00,               // Alignment
                            0x01,               // Length
                            )
                        IO (Decode16,
                            0x0064,             // Range Minimum
                            0x0064,             // Range Maximum
                            0x00,               // Alignment
                            0x01,               // Length
                            )
                        IRQNoFlags ()
                            {1}
                    })
                }

                Device (DMAC)
                {
                    Name (_HID, EisaId ("PNP0200") /* PC-class DMA Controller */)  // _HID: Hardware ID
                    Name (_CRS, ResourceTemplate ()  // _CRS: Current Resource Settings
                    {
                        IO (Decode16,
                            0x0000,             // Range Minimum
                            0x0000,             // Range Maximum
                            0x01,               // Alignment
                            0x10,               // Length
                            )
                        IO (Decode16,
                            0x0080,             // Range Minimum
                            0x0080,             // Range Maximum
                            0x01,               // Alignment
                            0x10,               // Length
                            )
                        IO (Decode16,
                            0x00C0,             // Range Minimum
                            0x00C0,             // Range Maximum
                            0x01,               // Alignment
                            0x20,               // Length
                            )
                        DMA (Compatibility, BusMaster, Transfer8_16, )
                            {4}
                    })
                }

                Device (FDC0)
                {
                    Name (_HID, EisaId ("PNP0700"))  // _HID: Hardware ID
                    Method (_STA, 0, NotSerialized)  // _STA: Status
                    {
                        Return (UFDC) /* \UFDC */
                    }

                    Name (_CRS, ResourceTemplate ()  // _CRS: Current Resource Settings
                    {
                        IO (Decode16,
                            0x03F0,             // Range Minimum
                            0x03F0,             // Range Maximum
                            0x01,               // Alignment
                            0x06,               // Length
                            )
                        IO (Decode16,
                            0x03F7,             // Range Minimum
                            0x03F7,             // Range Maximum
                            0x01,               // Alignment
                            0x01,               // Length
                            )
                        IRQNoFlags ()
                            {6}
                        DMA (Compatibility, NotBusMaster, Transfer8, )
                            {2}
                    })
                    Name (_PRS, ResourceTemplate ()  // _PRS: Possible Resource Settings
                    {
                        IO (Decode16,
                            0x03F0,             // Range Minimum
                            0x03F0,             // Range Maximum
                            0x01,               // Alignment
                            0x06,               // Length
                            )
                        IO (Decode16,
                            0x03F7,             // Range Minimum
                            0x03F7,             // Range Maximum
                            0x01,               // Alignment
                            0x01,               // Length
                            )
                        IRQNoFlags ()
                            {6}
                        DMA (Compatibility, NotBusMaster, Transfer8, )
                            {2}
                    })
                }

                Device (PS2M)
                {
                    Name (_HID, EisaId ("PNP0F03") /* Microsoft PS/2-style Mouse */)  // _HID: Hardware ID
                    Method (_STA, 0, NotSerialized)  // _STA: Status
                    {
                        Return (0x0F)
                    }

                    Name (_CRS, ResourceTemplate ()  // _CRS: Current Resource Settings
                    {
                        IRQNoFlags ()
                            {12}
                    })
                }

                Device (^LPT0)
                {
                    Name (_HID, EisaId ("PNP0400") /* Standard LPT Parallel Port */)  // _HID: Hardware ID
                    Name (_UID, One)  // _UID: Unique ID
                    Method (_STA, 0, NotSerialized)  // _STA: Status
                    {
                        If ((PP0B == Zero))
                        {
                            Return (Zero)
                        }
                        Else
                        {
                            Return (0x0F)
                        }
                    }

                    Name (CRS, ResourceTemplate ()
                    {
                        IO (Decode16,
                            0x0378,             // Range Minimum
                            0x0378,             // Range Maximum
                            0x08,               // Alignment
                            0x08,               // Length
                            _Y01)
                        IRQNoFlags (_Y02)
                            {7}
                    })
                    Method (_CRS, 0, NotSerialized)  // _CRS: Current Resource Settings
                    {
                        CreateWordField (CRS, \_SB.PCI0.LPT0._Y01._MIN, PMI0)  // _MIN: Minimum Base Address
                        CreateWordField (CRS, \_SB.PCI0.LPT0._Y01._MAX, PMA0)  // _MAX: Maximum Base Address
                        CreateWordField (CRS, \_SB.PCI0.LPT0._Y02._INT, PIQ0)  // _INT: Interrupts
                        PMI0 = PP0B /* \PP0B */
                        PMA0 = PP0B /* \PP0B */
                        PIQ0 = (One << PP0I) /* \PP0I */
                        Return (CRS) /* \_SB_.PCI0.LPT0.CRS_ */
                    }
                }

                Device (^LPT1)
                {
                    Name (_HID, EisaId ("PNP0400") /* Standard LPT Parallel Port */)  // _HID: Hardware ID
                    Name (_UID, 0x02)  // _UID: Unique ID
                    Method (_STA, 0, NotSerialized)  // _STA: Status
                    {
                        If ((PP1B == Zero))
                        {
                            Return (Zero)
                        }
                        Else
                        {
                            Return (0x0F)
                        }
                    }

                    Name (CRS, ResourceTemplate ()
                    {
                        IO (Decode16,
                            0x0278,             // Range Minimum
                            0x0278,             // Range Maximum
                            0x08,               // Alignment
                            0x08,               // Length
                            _Y03)
                        IRQNoFlags (_Y04)
                            {5}
                    })
                    Method (_CRS, 0, NotSerialized)  // _CRS: Current Resource Settings
                    {
                        CreateWordField (CRS, \_SB.PCI0.LPT1._Y03._MIN, PMI1)  // _MIN: Minimum Base Address
                        CreateWordField (CRS, \_SB.PCI0.LPT1._Y03._MAX, PMA1)  // _MAX: Maximum Base Address
                        CreateWordField (CRS, \_SB.PCI0.LPT1._Y04._INT, PIQ1)  // _INT: Interrupts
                        PMI1 = PP1B /* \PP1B */
                        PMA1 = PP1B /* \PP1B */
                        PIQ1 = (One << PP1I) /* \PP1I */
                        Return (CRS) /* \_SB_.PCI0.LPT1.CRS_ */
                    }
                }

                Device (^SRL0)
                {
                    Name (_HID, EisaId ("PNP0501") /* 16550A-compatible COM Serial Port */)  // _HID: Hardware ID
                    Name (_UID, One)  // _UID: Unique ID
                    Method (_STA, 0, NotSerialized)  // _STA: Status
                    {
                        If ((SL0B == Zero))
                        {
                            Return (Zero)
                        }
                        Else
                        {
                            Return (0x0F)
                        }
                    }

                    Name (CRS, ResourceTemplate ()
                    {
                        IO (Decode16,
                            0x03F8,             // Range Minimum
                            0x03F8,             // Range Maximum
                            0x01,               // Alignment
                            0x08,               // Length
                            _Y05)
                        IRQNoFlags (_Y06)
                            {4}
                    })
                    Method (_CRS, 0, NotSerialized)  // _CRS: Current Resource Settings
                    {
                        CreateWordField (CRS, \_SB.PCI0.SRL0._Y05._MIN, MIN0)  // _MIN: Minimum Base Address
                        CreateWordField (CRS, \_SB.PCI0.SRL0._Y05._MAX, MAX0)  // _MAX: Maximum Base Address
                        CreateWordField (CRS, \_SB.PCI0.SRL0._Y06._INT, IRQ0)  // _INT: Interrupts
                        MIN0 = SL0B /* \SL0B */
                        MAX0 = SL0B /* \SL0B */
                        IRQ0 = (One << SL0I) /* \SL0I */
                        Return (CRS) /* \_SB_.PCI0.SRL0.CRS_ */
                    }
                }

                Device (^SRL1)
                {
                    Name (_HID, EisaId ("PNP0501") /* 16550A-compatible COM Serial Port */)  // _HID: Hardware ID
                    Name (_UID, 0x02)  // _UID: Unique ID
                    Method (_STA, 0, NotSerialized)  // _STA: Status
                    {
                        If ((SL1B == Zero))
                        {
                            Return (Zero)
                        }
                        Else
                        {
                            Return (0x0F)
                        }
                    }

                    Name (CRS, ResourceTemplate ()
                    {
                        IO (Decode16,
                            0x02F8,             // Range Minimum
                            0x02F8,             // Range Maximum
                            0x01,               // Alignment
                            0x08,               // Length
                            _Y07)
                        IRQNoFlags (_Y08)
                            {3}
                    })
                    Method (_CRS, 0, NotSerialized)  // _CRS: Current Resource Settings
                    {
                        CreateWordField (CRS, \_SB.PCI0.SRL1._Y07._MIN, MIN1)  // _MIN: Minimum Base Address
                        CreateWordField (CRS, \_SB.PCI0.SRL1._Y07._MAX, MAX1)  // _MAX: Maximum Base Address
                        CreateWordField (CRS, \_SB.PCI0.SRL1._Y08._INT, IRQ1)  // _INT: Interrupts
                        MIN1 = SL1B /* \SL1B */
                        MAX1 = SL1B /* \SL1B */
                        IRQ1 = (One << SL1I) /* \SL1I */
                        Return (CRS) /* \_SB_.PCI0.SRL1.CRS_ */
                    }
                }

                Device (^SRL2)
                {
                    Name (_HID, EisaId ("PNP0501") /* 16550A-compatible COM Serial Port */)  // _HID: Hardware ID
                    Name (_UID, 0x03)  // _UID: Unique ID
                    Method (_STA, 0, NotSerialized)  // _STA: Status
                    {
                        If ((SL2B == Zero))
                        {
                            Return (Zero)
                        }
                        Else
                        {
                            Return (0x0F)
                        }
                    }

                    Name (CRS, ResourceTemplate ()
                    {
                        IO (Decode16,
                            0x03E8,             // Range Minimum
                            0x03E8,             // Range Maximum
                            0x01,               // Alignment
                            0x08,               // Length
                            _Y09)
                        IRQNoFlags (_Y0A)
                            {3}
                    })
                    Method (_CRS, 0, NotSerialized)  // _CRS: Current Resource Settings
                    {
                        CreateWordField (CRS, \_SB.PCI0.SRL2._Y09._MIN, MIN1)  // _MIN: Minimum Base Address
                        CreateWordField (CRS, \_SB.PCI0.SRL2._Y09._MAX, MAX1)  // _MAX: Maximum Base Address
                        CreateWordField (CRS, \_SB.PCI0.SRL2._Y0A._INT, IRQ1)  // _INT: Interrupts
                        MIN1 = SL2B /* \SL2B */
                        MAX1 = SL2B /* \SL2B */
                        IRQ1 = (One << SL2I) /* \SL2I */
                        Return (CRS) /* \_SB_.PCI0.SRL2.CRS_ */
                    }
                }

                Device (^SRL3)
                {
                    Name (_HID, EisaId ("PNP0501") /* 16550A-compatible COM Serial Port */)  // _HID: Hardware ID
                    Name (_UID, 0x04)  // _UID: Unique ID
                    Method (_STA, 0, NotSerialized)  // _STA: Status
                    {
                        If ((SL3B == Zero))
                        {
                            Return (Zero)
                        }
                        Else
                        {
                            Return (0x0F)
                        }
                    }

                    Name (CRS, ResourceTemplate ()
                    {
                        IO (Decode16,
                            0x02E8,             // Range Minimum
                            0x02E8,             // Range Maximum
                            0x01,               // Alignment
                            0x08,               // Length
                            _Y0B)
                        IRQNoFlags (_Y0C)
                            {3}
                    })
                    Method (_CRS, 0, NotSerialized)  // _CRS: Current Resource Settings
                    {
                        CreateWordField (CRS, \_SB.PCI0.SRL3._Y0B._MIN, MIN1)  // _MIN: Minimum Base Address
                        CreateWordField (CRS, \_SB.PCI0.SRL3._Y0B._MAX, MAX1)  // _MAX: Maximum Base Address
                        CreateWordField (CRS, \_SB.PCI0.SRL3._Y0C._INT, IRQ1)  // _INT: Interrupts
                        MIN1 = SL3B /* \SL3B */
                        MAX1 = SL3B /* \SL3B */
                        IRQ1 = (One << SL3I) /* \SL3I */
                        Return (CRS) /* \_SB_.PCI0.SRL3.CRS_ */
                    }
                }

                Device (TIMR)
                {
                    Name (_HID, EisaId ("PNP0100") /* PC-class System Timer */)  // _HID: Hardware ID
                    Name (_CRS, ResourceTemplate ()  // _CRS: Current Resource Settings
                    {
                        IO (Decode16,
                            0x0040,             // Range Minimum
                            0x0040,             // Range Maximum
                            0x00,               // Alignment
                            0x04,               // Length
                            )
                        IO (Decode16,
                            0x0050,             // Range Minimum
                            0x0050,             // Range Maximum
                            0x10,               // Alignment
                            0x04,               // Length
                            )
                    })
                }

                Device (PIC)
                {
                    Name (_HID, EisaId ("PNP0000") /* 8259-compatible Programmable Interrupt Controller */)  // _HID: Hardware ID
                    Name (_CRS, ResourceTemplate ()  // _CRS: Current Resource Settings
                    {
                        IO (Decode16,
                            0x0020,             // Range Minimum
                            0x0020,             // Range Maximum
                            0x00,               // Alignment
                            0x02,               // Length
                            )
                        IO (Decode16,
                            0x00A0,             // Range Minimum
                            0x00A0,             // Range Maximum
                            0x00,               // Alignment
                            0x02,               // Length
                            )
                        IRQNoFlags ()
                            {2}
                    })
                }

                Device (RTC)
                {
                    Name (_HID, EisaId ("PNP0B00") /* AT Real-Time Clock */)  // _HID: Hardware ID
                    Name (_CRS, ResourceTemplate ()  // _CRS: Current Resource Settings
                    {
                        IO (Decode16,
                            0x0070,             // Range Minimum
                            0x0070,             // Range Maximum
                            0x01,               // Alignment
                            0x02,               // Length
                            )
                    })
                    Method (_STA, 0, NotSerialized)  // _STA: Status
                    {
                        Return (URTC) /* \URTC */
                    }
                }

                Device (HPET)
                {
                    Name (_HID, EisaId ("PNP0103") /* HPET System Timer */)  // _HID: Hardware ID
                    Name (_CID, EisaId ("PNP0C01") /* System Board */)  // _CID: Compatible ID
                    Name (_UID, Zero)  // _UID: Unique ID
                    Method (_STA, 0, NotSerialized)  // _STA: Status
                    {
                        Return (UHPT) /* \UHPT */
                    }

                    Name (CRS, ResourceTemplate ()
                    {
                        IRQNoFlags ()
                            {0}
                        IRQNoFlags ()
                            {8}
                        Memory32Fixed (ReadWrite,
                            0xFED00000,         // Address Base
                            0x00000400,         // Address Length
                            )
                    })
                    Method (_CRS, 0, NotSerialized)  // _CRS: Current Resource Settings
                    {
                        Return (CRS) /* \_SB_.PCI0.SBRG.HPET.CRS_ */
                    }
                }

                Device (SMC)
                {
                    Name (_HID, EisaId ("APP0001"))  // _HID: Hardware ID
                    Name (_CID, "smc-napa")  // _CID: Compatible ID
                    Method (_STA, 0, NotSerialized)  // _STA: Status
                    {
                        Return (USMC) /* \USMC */
                    }

                    Name (CRS, ResourceTemplate ()
                    {
                        IO (Decode16,
                            0x0300,             // Range Minimum
                            0x0300,             // Range Maximum
                            0x01,               // Alignment
                            0x20,               // Length
                            )
                        IRQNoFlags ()
                            {6}
                    })
                    Method (_CRS, 0, NotSerialized)  // _CRS: Current Resource Settings
                    {
                        Return (CRS) /* \_SB_.PCI0.SBRG.SMC_.CRS_ */
                    }
                }
            }

            Device (GIGE)
            {
                Name (_HID, EisaId ("PNP8390"))  // _HID: Hardware ID
                Method (_ADR, 0, NotSerialized)  // _ADR: Address
                {
                    Return (NICA) /* \NICA */
                }

                Method (_STA, 0, NotSerialized)  // _STA: Status
                {
                    If ((NICA == Zero))
                    {
                        Return (Zero)
                    }
                    Else
                    {
                        Return (0x0F)
                    }
                }
            }

            Device (GFX0)
            {
                Name (_ADR, 0x00020000)  // _ADR: Address
                Method (_STA, 0, NotSerialized)  // _STA: Status
                {
                    If (((MSWN () > Zero) && (MSWN () < 0x08)))
                    {
                        Return (Zero)
                    }
                    Else
                    {
                        Return (0x0F)
                    }
                }

                Scope (\_GPE)
                {
                    Method (_L02, 0, NotSerialized)  // _Lxx: Level-Triggered GPE
                    {
                        Notify (\_SB.PCI0.GFX0, 0x81) // Information Change
                    }
                }

                Method (_DOS, 1, NotSerialized)  // _DOS: Disable Output Switching
                {
                }

                Method (_DOD, 0, NotSerialized)  // _DOD: Display Output Devices
                {
                    Return (Package (0x01)
                    {
                        0x80000100
                    })
                }

                Device (VGA)
                {
                    Method (_ADR, 0, Serialized)  // _ADR: Address
                    {
                        Return (0x0100)
                    }
                }
            }

            Device (HDEF)
            {
                Method (_DSM, 4, NotSerialized)  // _DSM: Device-Specific Method
                {
                    Local0 = Package (0x04)
                        {
                            "layout-id", 
                            Unicode ("\x04"), 
                            "PinConfigurations", 
                            Buffer (Zero) {}
                        }
                    If ((Arg0 == ToUUID ("a0b5b7c6-1318-441c-b0c9-fe695eaf949b")))
                    {
                        If ((Arg1 == One))
                        {
                            If ((Arg2 == Zero))
                            {
                                Local0 = Buffer (One)
                                    {
                                         0x03                                             /* . */
                                    }
                                Return (Local0)
                            }

                            If ((Arg2 == One))
                            {
                                Return (Local0)
                            }
                        }
                    }

                    Local0 = Buffer (One)
                        {
                             0x00                                             /* . */
                        }
                    Return (Local0)
                }

                Method (_ADR, 0, NotSerialized)  // _ADR: Address
                {
                    Return (HDAA) /* \HDAA */
                }

                Method (_STA, 0, NotSerialized)  // _STA: Status
                {
                    If ((HDAA == Zero))
                    {
                        Return (Zero)
                    }
                    Else
                    {
                        Return (0x0F)
                    }
                }
            }

            Device (BAT0)
            {
                Name (_HID, EisaId ("PNP0C0A") /* Control Method Battery */)  // _HID: Hardware ID
                Name (_UID, Zero)  // _UID: Unique ID
                Scope (\_GPE)
                {
                    Method (_L00, 0, NotSerialized)  // _Lxx: Level-Triggered GPE
                    {
                        Notify (\_SB.PCI0.BAT0, 0x80) // Status Change
                        Notify (\_SB.PCI0.AC, 0x80) // Status Change
                    }
                }

                OperationRegion (CBAT, SystemIO, 0x4040, 0x08)
                Field (CBAT, DWordAcc, NoLock, Preserve)
                {
                    IDX0,   32, 
                    DAT0,   32
                }

                IndexField (IDX0, DAT0, DWordAcc, NoLock, Preserve)
                {
                    STAT,   32, 
                    PRAT,   32, 
                    RCAP,   32, 
                    PVOL,   32, 
                    UNIT,   32, 
                    DCAP,   32, 
                    LFCP,   32, 
                    BTEC,   32, 
                    DVOL,   32, 
                    DWRN,   32, 
                    DLOW,   32, 
                    GRN1,   32, 
                    GRN2,   32, 
                    BSTA,   32, 
                    APSR,   32
                }

                Method (_STA, 0, NotSerialized)  // _STA: Status
                {
                    Return (BSTA) /* \_SB_.PCI0.BAT0.BSTA */
                }

                Name (PBIF, Package (0x0D)
                {
                    One, 
                    0x7FFFFFFF, 
                    0x7FFFFFFF, 
                    Zero, 
                    0xFFFFFFFF, 
                    Zero, 
                    Zero, 
                    0x04, 
                    0x04, 
                    "1", 
                    "0", 
                    "VBOX", 
                    "innotek"
                })
                Name (PBST, Package (0x04)
                {
                    Zero, 
                    0x7FFFFFFF, 
                    0x7FFFFFFF, 
                    0x7FFFFFFF
                })
                Method (_BIF, 0, NotSerialized)  // _BIF: Battery Information
                {
                    PBIF [Zero] = UNIT /* \_SB_.PCI0.BAT0.UNIT */
                    PBIF [One] = DCAP /* \_SB_.PCI0.BAT0.DCAP */
                    PBIF [0x02] = LFCP /* \_SB_.PCI0.BAT0.LFCP */
                    PBIF [0x03] = BTEC /* \_SB_.PCI0.BAT0.BTEC */
                    PBIF [0x04] = DVOL /* \_SB_.PCI0.BAT0.DVOL */
                    PBIF [0x05] = DWRN /* \_SB_.PCI0.BAT0.DWRN */
                    PBIF [0x06] = DLOW /* \_SB_.PCI0.BAT0.DLOW */
                    PBIF [0x07] = GRN1 /* \_SB_.PCI0.BAT0.GRN1 */
                    PBIF [0x08] = GRN2 /* \_SB_.PCI0.BAT0.GRN2 */
                    DBG ("_BIF:\n")
                    HEX4 (DerefOf (PBIF [Zero]))
                    HEX4 (DerefOf (PBIF [One]))
                    HEX4 (DerefOf (PBIF [0x02]))
                    HEX4 (DerefOf (PBIF [0x03]))
                    HEX4 (DerefOf (PBIF [0x04]))
                    HEX4 (DerefOf (PBIF [0x05]))
                    HEX4 (DerefOf (PBIF [0x06]))
                    HEX4 (DerefOf (PBIF [0x07]))
                    HEX4 (DerefOf (PBIF [0x08]))
                    Return (PBIF) /* \_SB_.PCI0.BAT0.PBIF */
                }

                Method (_BST, 0, NotSerialized)  // _BST: Battery Status
                {
                    PBST [Zero] = STAT /* \_SB_.PCI0.BAT0.STAT */
                    PBST [One] = PRAT /* \_SB_.PCI0.BAT0.PRAT */
                    PBST [0x02] = RCAP /* \_SB_.PCI0.BAT0.RCAP */
                    PBST [0x03] = PVOL /* \_SB_.PCI0.BAT0.PVOL */
                    Return (PBST) /* \_SB_.PCI0.BAT0.PBST */
                }
            }

            Device (AC)
            {
                Name (_HID, "ACPI0003" /* Power Source Device */)  // _HID: Hardware ID
                Name (_UID, Zero)  // _UID: Unique ID
                Name (_PCL, Package (0x01)  // _PCL: Power Consumer List
                {
                    _SB
                })
                Method (_PSR, 0, NotSerialized)  // _PSR: Power Source
                {
                    Return (^^BAT0.APSR) /* \_SB_.PCI0.BAT0.APSR */
                }

                Method (_STA, 0, NotSerialized)  // _STA: Status
                {
                    Return (0x0F)
                }
            }
        }
    }

    Scope (_SB)
    {
        Scope (PCI0)
        {
            Name (CRS, ResourceTemplate ()
            {
                WordBusNumber (ResourceProducer, MinFixed, MaxFixed, PosDecode,
                    0x0000,             // Granularity
                    0x0000,             // Range Minimum
                    0x00FF,             // Range Maximum
                    0x0000,             // Translation Offset
                    0x0100,             // Length
                    ,, )
                IO (Decode16,
                    0x0CF8,             // Range Minimum
                    0x0CF8,             // Range Maximum
                    0x01,               // Alignment
                    0x08,               // Length
                    )
                WordIO (ResourceProducer, MinFixed, MaxFixed, PosDecode, EntireRange,
                    0x0000,             // Granularity
                    0x0000,             // Range Minimum
                    0x0CF7,             // Range Maximum
                    0x0000,             // Translation Offset
                    0x0CF8,             // Length
                    ,, , TypeStatic)
                WordIO (ResourceProducer, MinFixed, MaxFixed, PosDecode, EntireRange,
                    0x0000,             // Granularity
                    0x0D00,             // Range Minimum
                    0xFFFF,             // Range Maximum
                    0x0000,             // Translation Offset
                    0xF300,             // Length
                    ,, , TypeStatic)
                DWordMemory (ResourceProducer, PosDecode, MinFixed, MaxFixed, Cacheable, ReadWrite,
                    0x00000000,         // Granularity
                    0x000A0000,         // Range Minimum
                    0x000BFFFF,         // Range Maximum
                    0x00000000,         // Translation Offset
                    0x00020000,         // Length
                    ,, , AddressRangeMemory, TypeStatic)
                DWordMemory (ResourceProducer, PosDecode, MinNotFixed, MaxFixed, Cacheable, ReadWrite,
                    0x00000000,         // Granularity
                    0x00000000,         // Range Minimum
                    0xFFDFFFFF,         // Range Maximum
                    0x00000000,         // Translation Offset
                    0x00000000,         // Length
                    ,, _Y0D, AddressRangeMemory, TypeStatic)
            })
            Name (TOM, ResourceTemplate ()
            {
                QWordMemory (ResourceProducer, PosDecode, MinFixed, MaxFixed, Prefetchable, ReadWrite,
                    0x0000000000000000, // Granularity
                    0x0000000100000000, // Range Minimum
                    0x0000000FFFFFFFFF, // Range Maximum
                    0x0000000000000000, // Translation Offset
                    0x0000000F00000000, // Length
                    ,, _Y0E, AddressRangeMemory, TypeStatic)
            })
            Method (_CRS, 0, NotSerialized)  // _CRS: Current Resource Settings
            {
                CreateDWordField (CRS, \_SB.PCI0._Y0D._MIN, RAMT)  // _MIN: Minimum Base Address
                CreateDWordField (CRS, \_SB.PCI0._Y0D._LEN, RAMR)  // _LEN: Length
                RAMT = MEML /* \MEML */
                RAMR = (0xFFE00000 - RAMT) /* \_SB_.PCI0._CRS.RAMT */
                If ((PMNN != Zero))
                {
                    If (((MSWN () < One) || (MSWN () > 0x06)))
                    {
                        CreateQWordField (TOM, \_SB.PCI0._Y0E._MIN, TM4N)  // _MIN: Minimum Base Address
                        CreateQWordField (TOM, \_SB.PCI0._Y0E._MAX, TM4X)  // _MAX: Maximum Base Address
                        CreateQWordField (TOM, \_SB.PCI0._Y0E._LEN, TM4L)  // _LEN: Length
                        TM4N = (PMNN * 0x00010000)
                        TM4X = ((PMNX * 0x00010000) - One)
                        TM4L = ((TM4X - TM4N) + One)
                        ConcatenateResTemplate (CRS, TOM, Local2)
                        Return (Local2)
                    }
                }

                Return (CRS) /* \_SB_.PCI0.CRS_ */
            }
        }
    }

    Scope (_SB)
    {
        Field (PCI0.SBRG.PCIC, ByteAcc, NoLock, Preserve)
        {
            Offset (0x60), 
            PIRA,   8, 
            PIRB,   8, 
            PIRC,   8, 
            PIRD,   8
        }

        Name (BUFA, ResourceTemplate ()
        {
            IRQ (Level, ActiveLow, Shared, )
                {15}
        })
        CreateWordField (BUFA, One, ICRS)
        Method (LSTA, 1, NotSerialized)
        {
            Local0 = (Arg0 & 0x80)
            If (Local0)
            {
                Return (0x09)
            }
            Else
            {
                Return (0x0B)
            }
        }

        Method (LCRS, 1, NotSerialized)
        {
            Local0 = (Arg0 & 0x0F)
            ICRS = (One << Local0)
            Return (BUFA) /* \_SB_.BUFA */
        }

        Method (LSRS, 1, NotSerialized)
        {
            CreateWordField (Arg0, One, ISRS)
            FindSetRightBit (ISRS, Local0)
            Return (Local0--)
        }

        Method (LDIS, 1, NotSerialized)
        {
            Return ((Arg0 | 0x80))
        }

        Device (LNKA)
        {
            Name (_HID, EisaId ("PNP0C0F") /* PCI Interrupt Link Device */)  // _HID: Hardware ID
            Name (_UID, One)  // _UID: Unique ID
            Method (_STA, 0, NotSerialized)  // _STA: Status
            {
                DBG ("LNKA._STA\n")
                Return (LSTA (PIRA))
            }

            Method (_PRS, 0, NotSerialized)  // _PRS: Possible Resource Settings
            {
                DBG ("LNKA._PRS\n")
                Return (PRSA) /* \_SB_.PRSA */
            }

            Method (_DIS, 0, NotSerialized)  // _DIS: Disable Device
            {
                DBG ("LNKA._DIS\n")
                PIRA = LDIS (PIRA)
            }

            Method (_CRS, 0, NotSerialized)  // _CRS: Current Resource Settings
            {
                DBG ("LNKA._CRS\n")
                Return (LCRS (PIRA))
            }

            Method (_SRS, 1, NotSerialized)  // _SRS: Set Resource Settings
            {
                DBG ("LNKA._SRS: ")
                HEX (LSRS (Arg0))
                PIRA = LSRS (Arg0)
            }
        }

        Device (LNKB)
        {
            Name (_HID, EisaId ("PNP0C0F") /* PCI Interrupt Link Device */)  // _HID: Hardware ID
            Name (_UID, 0x02)  // _UID: Unique ID
            Method (_STA, 0, NotSerialized)  // _STA: Status
            {
                Return (LSTA (PIRB))
            }

            Method (_PRS, 0, NotSerialized)  // _PRS: Possible Resource Settings
            {
                Return (PRSB) /* \_SB_.PRSB */
            }

            Method (_DIS, 0, NotSerialized)  // _DIS: Disable Device
            {
                PIRB = LDIS (PIRB)
            }

            Method (_CRS, 0, NotSerialized)  // _CRS: Current Resource Settings
            {
                Return (LCRS (PIRB))
            }

            Method (_SRS, 1, NotSerialized)  // _SRS: Set Resource Settings
            {
                DBG ("LNKB._SRS: ")
                HEX (LSRS (Arg0))
                PIRB = LSRS (Arg0)
            }
        }

        Device (LNKC)
        {
            Name (_HID, EisaId ("PNP0C0F") /* PCI Interrupt Link Device */)  // _HID: Hardware ID
            Name (_UID, 0x03)  // _UID: Unique ID
            Method (_STA, 0, NotSerialized)  // _STA: Status
            {
                Return (LSTA (PIRC))
            }

            Method (_PRS, 0, NotSerialized)  // _PRS: Possible Resource Settings
            {
                Return (PRSC) /* \_SB_.PRSC */
            }

            Method (_DIS, 0, NotSerialized)  // _DIS: Disable Device
            {
                PIRC = LDIS (PIRC)
            }

            Method (_CRS, 0, NotSerialized)  // _CRS: Current Resource Settings
            {
                Return (LCRS (PIRC))
            }

            Method (_SRS, 1, NotSerialized)  // _SRS: Set Resource Settings
            {
                DBG ("LNKC._SRS: ")
                HEX (LSRS (Arg0))
                PIRC = LSRS (Arg0)
            }
        }

        Device (LNKD)
        {
            Name (_HID, EisaId ("PNP0C0F") /* PCI Interrupt Link Device */)  // _HID: Hardware ID
            Name (_UID, 0x04)  // _UID: Unique ID
            Method (_STA, 0, NotSerialized)  // _STA: Status
            {
                Return (LSTA (PIRD))
            }

            Method (_PRS, 0, NotSerialized)  // _PRS: Possible Resource Settings
            {
                Return (PRSD) /* \_SB_.PRSD */
            }

            Method (_DIS, 0, NotSerialized)  // _DIS: Disable Device
            {
                PIRD = LDIS (PIRA)
            }

            Method (_CRS, 0, NotSerialized)  // _CRS: Current Resource Settings
            {
                Return (LCRS (PIRD))
            }

            Method (_SRS, 1, NotSerialized)  // _SRS: Set Resource Settings
            {
                DBG ("LNKD._SRS: ")
                HEX (LSRS (Arg0))
                PIRD = LSRS (Arg0)
            }
        }
    }

    Name (_S0, Package (0x02)  // _S0_: S0 System State
    {
        Zero, 
        Zero
    })
    If ((PWRS & 0x02))
    {
        Name (_S1, Package (0x02)  // _S1_: S1 System State
        {
            One, 
            One
        })
    }

    If ((PWRS & 0x10))
    {
        Name (_S4, Package (0x02)  // _S4_: S4 System State
        {
            0x05, 
            0x05
        })
    }

    Name (_S5, Package (0x02)  // _S5_: S5 System State
    {
        0x05, 
        0x05
    })
    Method (_PTS, 1, NotSerialized)  // _PTS: Prepare To Sleep
    {
        DBG ("Prepare to sleep: ")
        HEX (Arg0)
    }
}

