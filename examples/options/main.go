// Example: Custom Options
//
// This example demonstrates the various configuration options
// available for chunking code.
package main

import (
	"fmt"
	"log"
	"strings"

	codechunk "github.com/pc-coder/go-code-chunk"
)

func main() {
	// Sample TypeScript code
	code := `import { useState, useEffect, useCallback } from 'react';
import { User, fetchUser, updateUser } from './api';

interface UserProfile {
	id: string;
	name: string;
	email: string;
	avatar?: string;
}

interface UserProfileProps {
	userId: string;
	onUpdate?: (user: UserProfile) => void;
}

/**
 * UserProfile component displays and allows editing of user information.
 * @param props Component properties
 */
function UserProfile({ userId, onUpdate }: UserProfileProps): JSX.Element {
	const [user, setUser] = useState<UserProfile | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		async function loadUser() {
			try {
				const userData = await fetchUser(userId);
				setUser(userData);
			} catch (e) {
				setError('Failed to load user');
			} finally {
				setLoading(false);
			}
		}
		loadUser();
	}, [userId]);

	const handleUpdate = useCallback(async (updates: Partial<UserProfile>) => {
		if (!user) return;
		try {
			const updated = await updateUser(user.id, updates);
			setUser(updated);
			onUpdate?.(updated);
		} catch (e) {
			setError('Failed to update user');
		}
	}, [user, onUpdate]);

	if (loading) return <div>Loading...</div>;
	if (error) return <div>Error: {error}</div>;
	if (!user) return <div>User not found</div>;

	return (
		<div className="user-profile">
			<img src={user.avatar || '/default-avatar.png'} alt={user.name} />
			<h2>{user.name}</h2>
			<p>{user.email}</p>
			<button onClick={() => handleUpdate({ name: 'Updated Name' })}>
				Update Name
			</button>
		</div>
	);
}

export default UserProfile;
`

	// Example 1: Default options
	fmt.Println("=== Example 1: Default Options ===")
	chunks1, err := codechunk.Chunk("UserProfile.tsx", code, nil)
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	printChunkSummary("Default", chunks1)

	// Example 2: Small chunk size
	fmt.Println("\n=== Example 2: Small Chunk Size ===")
	chunks2, err := codechunk.Chunk("UserProfile.tsx", code, &codechunk.ChunkOptions{
		MaxChunkSize: 300,
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	printChunkSummary("SmallChunks", chunks2)

	// Example 3: No context
	fmt.Println("\n=== Example 3: No Context Mode ===")
	chunks3, err := codechunk.Chunk("UserProfile.tsx", code, &codechunk.ChunkOptions{
		ContextMode: codechunk.ContextModeNone,
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	printChunkSummary("NoContext", chunks3)
	// Show that context is empty
	if len(chunks3) > 0 {
		fmt.Printf("  Scope items: %d\n", len(chunks3[0].Context.Scope))
		fmt.Printf("  Entity items: %d\n", len(chunks3[0].Context.Entities))
	}

	// Example 4: Minimal context
	fmt.Println("\n=== Example 4: Minimal Context Mode ===")
	chunks4, err := codechunk.Chunk("UserProfile.tsx", code, &codechunk.ChunkOptions{
		ContextMode: codechunk.ContextModeMinimal,
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	printChunkSummary("MinimalContext", chunks4)

	// Example 5: Filter imports
	fmt.Println("\n=== Example 5: Filtered Imports ===")
	chunks5, err := codechunk.Chunk("UserProfile.tsx", code, &codechunk.ChunkOptions{
		FilterImports: true,
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	printChunkSummary("FilteredImports", chunks5)
	if len(chunks5) > 0 {
		fmt.Println("  Imports:")
		for _, imp := range chunks5[0].Context.Imports {
			fmt.Printf("    - %s (from: %s)\n", imp.Name, imp.Source)
		}
	}

	// Example 6: No siblings
	fmt.Println("\n=== Example 6: No Siblings ===")
	chunks6, err := codechunk.Chunk("UserProfile.tsx", code, &codechunk.ChunkOptions{
		SiblingDetail: codechunk.SiblingDetailNone,
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	printChunkSummary("NoSiblings", chunks6)
	if len(chunks6) > 0 {
		fmt.Printf("  Sibling count: %d\n", len(chunks6[0].Context.Siblings))
	}

	// Example 7: Names only siblings
	fmt.Println("\n=== Example 7: Sibling Names Only ===")
	chunks7, err := codechunk.Chunk("UserProfile.tsx", code, &codechunk.ChunkOptions{
		SiblingDetail: codechunk.SiblingDetailNames,
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	printChunkSummary("SiblingNames", chunks7)

	// Example 8: Large overlap
	fmt.Println("\n=== Example 8: Large Overlap ===")
	chunks8, err := codechunk.Chunk("UserProfile.tsx", code, &codechunk.ChunkOptions{
		MaxChunkSize: 400,
		OverlapLines: 20,
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	printChunkSummary("LargeOverlap", chunks8)

	// Example 9: Force language
	fmt.Println("\n=== Example 9: Force Language ===")
	// Process .tsx file as JavaScript instead of TypeScript
	chunks9, err := codechunk.Chunk("UserProfile.tsx", code, &codechunk.ChunkOptions{
		Language: codechunk.LanguageJavaScript,
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	printChunkSummary("ForcedJS", chunks9)
	if len(chunks9) > 0 {
		fmt.Printf("  Detected language: %s\n", chunks9[0].Context.Language)
	}

	// Example 10: Using Chunker instance
	fmt.Println("\n=== Example 10: Reusable Chunker ===")
	chunker := codechunk.NewChunker(&codechunk.ChunkOptions{
		MaxChunkSize:  800,
		ContextMode:   codechunk.ContextModeFull,
		SiblingDetail: codechunk.SiblingDetailSignatures,
		OverlapLines:  10,
	})

	// Use with defaults
	chunks10a, err := chunker.Chunk("UserProfile.tsx", code, nil)
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	printChunkSummary("ChunkerDefaults", chunks10a)

	// Override specific option
	chunks10b, err := chunker.Chunk("UserProfile.tsx", code, &codechunk.ChunkOptions{
		MaxChunkSize: 400, // Override just this
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	printChunkSummary("ChunkerOverride", chunks10b)

	// Example 11: Show contextualized text format
	fmt.Println("\n=== Example 11: Contextualized Text Format ===")
	chunks11, err := codechunk.Chunk("UserProfile.tsx", code, &codechunk.ChunkOptions{
		MaxChunkSize: 600,
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	if len(chunks11) > 0 {
		fmt.Println("First chunk's contextualized text:")
		fmt.Println(strings.Repeat("-", 50))
		// Show first 1000 chars
		text := chunks11[0].ContextualizedText
		if len(text) > 1000 {
			text = text[:1000] + "..."
		}
		fmt.Println(text)
		fmt.Println(strings.Repeat("-", 50))
	}
}

func printChunkSummary(name string, chunks []codechunk.CodeChunk) {
	totalBytes := 0
	totalEntities := 0
	for _, c := range chunks {
		totalBytes += len(c.Text)
		totalEntities += len(c.Context.Entities)
	}

	fmt.Printf("  %s: %d chunks, %d total bytes, %d entities\n",
		name, len(chunks), totalBytes, totalEntities)
}
