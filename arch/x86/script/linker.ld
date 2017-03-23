ENTRY(_rt0_entry)

SECTIONS {
	/* Kernel starts at 1M */
	. = 1M;

	/* ensure that the multiboot header is at the beginning */
	.multiboot :
	{
		*(.multiboot_header)
	}

	.text BLOCK(4K) : ALIGN(4K)
	{
		*(.text)
	}

	/* Read-only data. */
	.rodata BLOCK(4K) : ALIGN(4K)
	{
		*(.rodata)
	}

	/* Read-write data (initialized) */
	.data BLOCK(4K) : ALIGN(4K)
	{
		*(.data)
	}

	.bss BLOCK(4K) : ALIGN(4K)
	{
		*(COMMON)
		*(.bss)
	}
}
