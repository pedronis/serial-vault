// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016-2017 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package crypt

import "testing"

func TestEncryptDecrypt(t *testing.T) {

	plainText := "fake-hmac-ed-data"

	cipherText, err := EncryptKey(plainText, "this needs to be 32 bytes long!!")
	if err != nil {
		t.Errorf("Error encrypting text: %v", err)
	}
	if string(cipherText[:]) == plainText {
		t.Error("Invalid encryption")
	}

	plainTextAgain, err := DecryptKey(cipherText, "this needs to be 32 bytes long!!")
	if err != nil {
		t.Errorf("Error decrypting text: %v", err)
	}
	if string(plainTextAgain[:]) != plainText {
		t.Error("Invalid decryption")
	}
}
